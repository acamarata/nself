// Package auth — HTTP client for nSelf auth server operations.
package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// AuthServerURL is the base URL of the auth server.
// Overridable via NSELF_AUTH_SERVER_URL for testing.
func AuthServerURL() string {
	if url := getEnv("NSELF_AUTH_SERVER_URL"); url != "" {
		return url
	}
	return "https://api.nself.org"
}

// CLIAuthBaseURL is the web UI for the CLI device auth page.
func CLIAuthBaseURL() string {
	if url := getEnv("NSELF_CLI_AUTH_URL"); url != "" {
		return url
	}
	return "https://nself.org/auth/cli"
}

// DeviceCodeResponse is returned by the device authorization endpoint.
type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`        // XXXX-YYYY format shown to user
	VerificationURL string `json:"verification_url"` // nself.org/auth/cli?code=...
	ExpiresInSec    int    `json:"expires_in"`
	IntervalSec     int    `json:"interval"`
}

// TokenResponse is returned when the device code is exchanged for a token.
type TokenResponse struct {
	AccessToken  string   `json:"access_token"`
	SessionToken string   `json:"session_token"`
	Email        string   `json:"email"`
	Tier         string   `json:"tier"`
	DisplayName  string   `json:"display_name,omitempty"`
	Bundles      []string `json:"bundles,omitempty"`
	ExpiresAt    string   `json:"expires_at"`
}

// AccountInfo is the response from GET /auth/session.
type AccountInfo struct {
	Authenticated bool `json:"authenticated"`
	Account       struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		DisplayName   string `json:"display_name"`
		AvatarURL     string `json:"avatar_url"`
		Tier          string `json:"tier"`
		EmailVerified bool   `json:"email_verified"`
		MFAEnabled    bool   `json:"mfa_enabled"`
	} `json:"account"`
}

// LicenseInfo represents a single license entry.
type LicenseInfo struct {
	ID            string   `json:"id"`
	Product       string   `json:"product"`
	Tier          string   `json:"tier"`
	Bundles       []string `json:"bundles"`
	SeatsIncluded int      `json:"seats_included"`
	SeatsUsed     int      `json:"seats_used"`
	IsActive      bool     `json:"is_active"`
	ActivatedAt   string   `json:"activated_at"`
	ExpiresAt     string   `json:"expires_at"`
}

// AuthAPIError is returned when the auth server returns a non-2xx response.
type AuthAPIError struct {
	Code    string `json:"error"`
	Message string `json:"message"`
	Status  int
}

func (e *AuthAPIError) Error() string {
	return fmt.Sprintf("auth server error [%s]: %s (HTTP %d)", e.Code, e.Message, e.Status)
}

// httpClient is the shared HTTP client with timeout.
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

// DeviceAuthorize initiates the device code flow.
// Returns a DeviceCodeResponse with the code to display to the user.
func DeviceAuthorize(ctx context.Context) (*DeviceCodeResponse, error) {
	url := fmt.Sprintf("%s/auth/device/authorize", AuthServerURL())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBufferString(`{}`))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("contacting auth server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, parseAPIError(resp)
	}

	var result DeviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing device authorization response: %w", err)
	}

	return &result, nil
}

// PollToken polls the auth server for the device code exchange result.
// Returns (nil, nil) if the user hasn't authorized yet (authorization_pending).
// Returns an error on timeout or other failure.
func PollToken(ctx context.Context, deviceCode string) (*TokenResponse, error) {
	url := fmt.Sprintf("%s/auth/device/token", AuthServerURL())

	body, _ := json.Marshal(map[string]string{"device_code": deviceCode})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("polling token: %w", err)
	}
	defer resp.Body.Close()

	// authorization_pending → user hasn't clicked "Authorize" yet
	if resp.StatusCode == http.StatusAccepted {
		return nil, nil
	}

	if resp.StatusCode == http.StatusOK {
		var result TokenResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("parsing token response: %w", err)
		}
		return &result, nil
	}

	return nil, parseAPIError(resp)
}

// RefreshToken exchanges an existing session for a new access token.
func RefreshToken(ctx context.Context, accessToken string) (*TokenResponse, error) {
	url := fmt.Sprintf("%s/auth/refresh", AuthServerURL())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cookie", fmt.Sprintf("nself_auth_token=%s", accessToken))

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("refreshing token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var result TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing refresh response: %w", err)
	}

	return &result, nil
}

// RevokeSession calls POST /auth/signout to revoke the session server-side.
func RevokeSession(ctx context.Context, accessToken string, all bool) error {
	url := fmt.Sprintf("%s/auth/signout", AuthServerURL())
	if all {
		url += "?all=true"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Cookie", fmt.Sprintf("nself_auth_token=%s", accessToken))

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("revoking session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return parseAPIError(resp)
	}

	return nil
}

// GetSession returns the account info for the given access token.
func GetSession(ctx context.Context, accessToken string) (*AccountInfo, error) {
	url := fmt.Sprintf("%s/auth/session", AuthServerURL())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cookie", fmt.Sprintf("nself_auth_token=%s", accessToken))

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("getting session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrNotLoggedIn
	}

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var result AccountInfo
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing session response: %w", err)
	}

	return &result, nil
}

// GetLicenses returns the list of active licenses for the account.
func GetLicenses(ctx context.Context, accessToken string) ([]LicenseInfo, error) {
	url := fmt.Sprintf("%s/account/licenses", AuthServerURL())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cookie", fmt.Sprintf("nself_auth_token=%s", accessToken))

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("getting licenses: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var result struct {
		Licenses []LicenseInfo `json:"licenses"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing licenses response: %w", err)
	}

	return result.Licenses, nil
}

// ─── Team ─────────────────────────────────────────────────────────────────────

// TeamMember represents one member of an account's team.
type TeamMember struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	JoinedAt string `json:"joined_at"`
}

// GetTeamMembers returns the list of team members for the account.
func GetTeamMembers(ctx context.Context, accessToken string) ([]TeamMember, error) {
	url := fmt.Sprintf("%s/account/team", AuthServerURL())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cookie", fmt.Sprintf("nself_auth_token=%s", accessToken))

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("getting team: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var result struct {
		Members []TeamMember `json:"members"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing team response: %w", err)
	}
	return result.Members, nil
}

// InviteTeamMember sends a team invitation to the given email.
func InviteTeamMember(ctx context.Context, accessToken, email string) error {
	url := fmt.Sprintf("%s/account/team/invite", AuthServerURL())
	body, _ := json.Marshal(map[string]string{"email": email})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Cookie", fmt.Sprintf("nself_auth_token=%s", accessToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("inviting team member: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return parseAPIError(resp)
	}
	return nil
}

// RemoveTeamMember removes a member from the account's team.
func RemoveTeamMember(ctx context.Context, accessToken, email string) error {
	url := fmt.Sprintf("%s/account/team/%s", AuthServerURL(), email)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Cookie", fmt.Sprintf("nself_auth_token=%s", accessToken))

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("removing team member: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return parseAPIError(resp)
	}
	return nil
}

// SetTeamMemberRole updates a member's role on the account's team.
func SetTeamMemberRole(ctx context.Context, accessToken, email, role string) error {
	url := fmt.Sprintf("%s/account/team/%s/role", AuthServerURL(), email)
	body, _ := json.Marshal(map[string]string{"role": role})
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Cookie", fmt.Sprintf("nself_auth_token=%s", accessToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("setting team member role: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return parseAPIError(resp)
	}
	return nil
}

// ─── Licenses (extended) ──────────────────────────────────────────────────────

// ActivateLicense activates a license key on the current device.
func ActivateLicense(ctx context.Context, accessToken, licenseID string) error {
	url := fmt.Sprintf("%s/account/licenses/%s/activate", AuthServerURL(), licenseID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Cookie", fmt.Sprintf("nself_auth_token=%s", accessToken))

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("activating license: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return parseAPIError(resp)
	}
	return nil
}

// ─── Devices ──────────────────────────────────────────────────────────────────

// DeviceEntry represents one registered device for an account.
type DeviceEntry struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	OS         string `json:"os"`
	LastActive string `json:"last_active"`
	IsCurrent  bool   `json:"is_current"`
}

// GetDevices returns the list of registered devices for the account.
func GetDevices(ctx context.Context, accessToken string) ([]DeviceEntry, error) {
	url := fmt.Sprintf("%s/account/devices", AuthServerURL())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cookie", fmt.Sprintf("nself_auth_token=%s", accessToken))

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("getting devices: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var result struct {
		Devices []DeviceEntry `json:"devices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing devices response: %w", err)
	}
	return result.Devices, nil
}

// RevokeDevice revokes a specific device session.
func RevokeDevice(ctx context.Context, accessToken, deviceID string) error {
	url := fmt.Sprintf("%s/account/devices/%s", AuthServerURL(), deviceID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Cookie", fmt.Sprintf("nself_auth_token=%s", accessToken))

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("revoking device: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return parseAPIError(resp)
	}
	return nil
}

// ─── Transfer ─────────────────────────────────────────────────────────────────

// TransferLicense transfers a license from the current account to another email.
func TransferLicense(ctx context.Context, accessToken, licenseID, toEmail string) error {
	url := fmt.Sprintf("%s/account/licenses/%s/transfer", AuthServerURL(), licenseID)
	body, _ := json.Marshal(map[string]string{"to_email": toEmail})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Cookie", fmt.Sprintf("nself_auth_token=%s", accessToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("transferring license: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return parseAPIError(resp)
	}
	return nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func parseAPIError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var apiErr AuthAPIError
	if err := json.Unmarshal(body, &apiErr); err != nil {
		return fmt.Errorf("auth server error (HTTP %d): %s", resp.StatusCode, string(body))
	}
	apiErr.Status = resp.StatusCode
	return &apiErr
}

func getEnv(key string) string {
	// os imported at package level via storage.go
	return os.Getenv(key)
}

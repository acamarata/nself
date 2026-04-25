package watchdog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/smtp"
	"os"
	"time"
)

// Incident records a watchdog incident for persistence.
type Incident struct {
	ID        string    `json:"id"`
	Service   string    `json:"service"`
	Severity  string    `json:"severity"` // warning, critical
	Action    string    `json:"action"`
	Detail    string    `json:"detail"`
	Timestamp time.Time `json:"timestamp"`
	Notified  bool      `json:"notified"`
}

// EscalationConfig holds TG and SMTP notification settings.
type EscalationConfig struct {
	TelegramBotToken string
	TelegramChatID   string
	SMTPHost         string
	SMTPPort         string
	SMTPFrom         string
	SMTPTo           string
	SMTPUser         string
	SMTPPass         string
}

// LoadEscalationConfig reads escalation config from environment variables.
func LoadEscalationConfig() EscalationConfig {
	return EscalationConfig{
		TelegramBotToken: os.Getenv("WATCHDOG_TG_BOT_TOKEN"),
		TelegramChatID:   os.Getenv("WATCHDOG_TG_CHAT_ID"),
		SMTPHost:         os.Getenv("WATCHDOG_SMTP_HOST"),
		SMTPPort:         getEnvDefault("WATCHDOG_SMTP_PORT", "587"),
		SMTPFrom:         os.Getenv("WATCHDOG_SMTP_FROM"),
		SMTPTo:           getEnvDefault("WATCHDOG_SMTP_TO", "ops@nself.org"),
		SMTPUser:         os.Getenv("WATCHDOG_SMTP_USER"),
		SMTPPass:         os.Getenv("WATCHDOG_SMTP_PASS"),
	}
}

func getEnvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// SendTelegramAlert sends an alert message to a Telegram chat.
func SendTelegramAlert(cfg EscalationConfig, message string) error {
	if cfg.TelegramBotToken == "" || cfg.TelegramChatID == "" {
		return fmt.Errorf("Telegram bot token or chat ID not configured")
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cfg.TelegramBotToken)
	body := map[string]string{
		"chat_id":    cfg.TelegramChatID,
		"text":       message,
		"parse_mode": "Markdown",
	}
	data, _ := json.Marshal(body)

	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("sending Telegram alert: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Telegram API returned %d", resp.StatusCode)
	}
	return nil
}

// SendEmailAlert sends an alert email via SMTP. Uses a simple net/smtp approach.
func SendEmailAlert(cfg EscalationConfig, subject, body string) error {
	if cfg.SMTPHost == "" || cfg.SMTPFrom == "" || cfg.SMTPTo == "" {
		return fmt.Errorf("SMTP not configured (missing host, from, or to)")
	}

	// Send via SMTP (STARTTLS on port 587, plain auth).
	addr := fmt.Sprintf("%s:%s", cfg.SMTPHost, cfg.SMTPPort)
	auth := smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPHost)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		cfg.SMTPFrom, cfg.SMTPTo, subject, body)
	if err := smtp.SendMail(addr, auth, cfg.SMTPFrom, []string{cfg.SMTPTo}, []byte(msg)); err != nil {
		slog.Warn("watchdog: SMTP send failed", "err", err)
		return fmt.Errorf("smtp send: %w", err)
	}
	return nil
}

// Escalate sends alerts through all configured channels.
func Escalate(service, severity, detail string) []error {
	cfg := LoadEscalationConfig()
	var errs []error

	message := fmt.Sprintf("*[%s] Watchdog Alert: %s*\nService: `%s`\nDetail: %s\nTime: %s",
		severity, service, service, detail, time.Now().UTC().Format(time.RFC3339))

	if err := SendTelegramAlert(cfg, message); err != nil {
		errs = append(errs, fmt.Errorf("telegram: %w", err))
	}

	subject := fmt.Sprintf("[%s] Watchdog: %s — %s", severity, service, detail)
	if err := SendEmailAlert(cfg, subject, message); err != nil {
		errs = append(errs, fmt.Errorf("email: %w", err))
	}

	return errs
}

// TestAlert sends a test alert through all configured channels.
// Returns errors for any channel that failed.
func TestAlert(service, severity string) ([]string, []error) {
	cfg := LoadEscalationConfig()
	var delivered []string
	var errs []error

	detail := fmt.Sprintf("Test alert for service=%s severity=%s", service, severity)
	message := fmt.Sprintf("*[TEST] Watchdog Alert*\nService: `%s`\nSeverity: %s\nDetail: %s\nTime: %s",
		service, severity, detail, time.Now().UTC().Format(time.RFC3339))

	if err := SendTelegramAlert(cfg, message); err != nil {
		errs = append(errs, fmt.Errorf("telegram: %w", err))
	} else {
		delivered = append(delivered, "telegram")
	}

	subject := fmt.Sprintf("[TEST][%s] Watchdog: %s", severity, service)
	if err := SendEmailAlert(cfg, subject, message); err != nil {
		errs = append(errs, fmt.Errorf("email: %w", err))
	} else {
		delivered = append(delivered, "email")
	}

	return delivered, errs
}

# Plugin: auth-enterprise

**Port:** 3826 · **Tier:** max (ɳSelf+) · **Category:** authentication

The `auth-enterprise` plugin adds enterprise-grade authentication to any nSelf
deployment: MFA enforcement via TOTP and WebAuthn policy, and Single Sign-On via
SAML 2.0 and OIDC. MFA is always enabled per the Security-Always-Free doctrine — it
cannot be disabled by operators or users.

---

## Install

```bash
nself plugin install auth-enterprise
```

The plugin downloads, verifies the checksum, and starts on port 3826.

---

## Environment variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | Yes | — | PostgreSQL connection string |
| `AUTH_ENTERPRISE_TOTP_ISSUER` | No | `nSelf` | Label shown in authenticator apps |
| `NSELF_SSO` | No | `false` | Set `true` to activate SSO endpoints |
| `AUTH_ENTERPRISE_SSO_SP_ENTITY_ID` | When SSO | — | SAML SP entity ID URI |
| `AUTH_ENTERPRISE_SSO_ACS_URL` | When SSO | — | SAML Assertion Consumer Service URL |
| `AUTH_ENTERPRISE_SSO_OIDC_CALLBACK_URL` | When SSO | — | OIDC redirect URI |

---

## MFA — TOTP setup

TOTP follows RFC 6238 (6 digits, 30-second window, ±1 step drift tolerance).

**Enrollment flow:**

1. Call `POST /auth/mfa/totp/setup` — returns a QR-code-ready `otp_uri` and eight
   single-use backup codes. Show both to the user once; they are not stored in plaintext.
2. User scans the QR code with an authenticator app (Google Authenticator, Authy, etc.).
3. Call `POST /auth/mfa/totp/verify` with a live 6-digit code to activate the enrollment.
4. On each login, call `POST /auth/mfa/totp/challenge` to verify the user's code.

**Recovery codes** are bcrypt-hashed (cost 10) and consumed one-at-a-time via
`POST /auth/mfa/recovery`. Replace all codes with `POST /auth/mfa/recovery/regenerate`.

---

## SSO — Google Workspace (OIDC)

Enable with `NSELF_SSO=true`.

1. Create an OAuth 2.0 client in Google Cloud Console (APIs & Services → Credentials).
2. Set the **Authorized redirect URI** to your `AUTH_ENTERPRISE_SSO_OIDC_CALLBACK_URL`.
3. Create the provider in nSelf:
   ```bash
   curl -X POST https://your-nself-host/auth/sso/providers \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"protocol":"oidc","idp_name":"google","metadata":{"client_id":"...","client_secret":"..."}}'
   ```
4. The OIDC authorization code flow starts at `GET /auth/sso/oidc/{provider_id}/begin`.
   JIT provisioning creates the nSelf user row automatically on first login.

---

## SSO — Okta (SAML 2.0)

Enable with `NSELF_SSO=true`.

1. In Okta Admin → Applications, create a SAML 2.0 app.
2. Set **Single sign-on URL** (ACS URL) to `AUTH_ENTERPRISE_SSO_ACS_URL`.
3. Set **Audience URI (SP Entity ID)** to `AUTH_ENTERPRISE_SSO_SP_ENTITY_ID`.
4. Download the IdP signing certificate (PEM, no headers).
5. Create the provider:
   ```bash
   curl -X POST https://your-nself-host/auth/sso/providers \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"protocol":"saml","idp_name":"okta","metadata":{"sso_url":"https://yourorg.okta.com/app/.../sso/saml","entity_id":"http://www.okta.com/...","certificate":"<base64-pem>"}}'
   ```
6. Upload SP metadata back to Okta:
   ```bash
   curl https://your-nself-host/auth/sso/providers/{id}/saml/metadata \
     -H "Authorization: Bearer $TOKEN"
   ```

---

## SSO — Microsoft Entra ID (SAML 2.0)

Enable with `NSELF_SSO=true`.

1. In Azure → Enterprise Applications → New application → Create your own.
2. Under **Single sign-on → SAML**:
   - **Reply URL (ACS):** set to `AUTH_ENTERPRISE_SSO_ACS_URL`
   - **Identifier (Entity ID):** set to `AUTH_ENTERPRISE_SSO_SP_ENTITY_ID`
3. In **SAML Certificates**, download the **Certificate (Base64)**.
4. Copy the **Login URL** (SSO URL) and **Azure AD Identifier** (Entity ID).
5. Create the provider:
   ```bash
   curl -X POST https://your-nself-host/auth/sso/providers \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"protocol":"saml","idp_name":"entra","metadata":{"sso_url":"https://login.microsoftonline.com/.../saml2","entity_id":"https://sts.windows.net/.../","certificate":"<base64-cert>"}}'
   ```

---

## MFA policy

Control enforcement per tenant via the policy API:

```bash
# Set policy (requires mfa:admin scope)
curl -X PUT https://your-nself-host/auth/mfa/policy \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"totp_required":true,"webauthn_mode":"optional"}'

# Get current policy
curl https://your-nself-host/auth/mfa/policy \
  -H "Authorization: Bearer $TOKEN"
```

`webauthn_mode` accepts `required`, `optional`, or `disabled`.

---

## Database tables

All tables carry `source_account_id TEXT NOT NULL DEFAULT 'primary'` for multi-app
isolation.

| Table | Purpose |
|---|---|
| `np_mfa_enrollments` | TOTP secret + state per user |
| `np_mfa_recovery_codes` | Bcrypt-hashed single-use backup codes |
| `np_mfa_policies` | Per-tenant enforcement settings |
| `np_sso_providers` | SAML/OIDC IdP configurations |
| `np_sso_sessions` | Active SSO sessions |
| `np_sso_state_cache` | SAML relay-state + OIDC PKCE nonces |

---

## Security notes

- TOTP secrets are stored base32 in `np_mfa_enrollments.totp_secret`. Enable
  pgcrypto column encryption via `NSELF_MFA_ENCRYPTION_KEY` for at-rest protection.
- Recovery codes are bcrypt cost-10 hashed; raw codes are returned once at enrollment
  and never stored in plaintext.
- SSO `client_secret` values should be encrypted before writing to `np_sso_providers`.
- SAML assertion signature verification enforces audience and NameID presence; full
  XMLDSig validation requires integrating `crewjam/saml` at the handler boundary.
- MFA is always on regardless of `NSELF_SSO` (Security-Always-Free doctrine).
  SSO requires `NSELF_SSO=true` and a valid ɳSelf+ license entitlement.

---

## Docker Hub

```bash
docker pull nself-org/plugin-auth-enterprise:latest
```

---

## See also

- [`nself plugin install`](commands/plugin-install.md)
- [Security-Always-Free doctrine](security/security-always-free.md)
- [MFA + auth plugin](plugin-auth.md) — the free MFA enforcement layer (port 3821)

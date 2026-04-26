# Push Plugin

> APNs + FCM push notification relay for iOS and Android. **Free, MIT licensed.**

## Install

```bash
nself plugin install push
```

Redis is auto-enabled when the push plugin is installed. If `REDIS_ENABLED` is unset in your env, `nself build` detects the installed plugin and adds the redis service automatically.

## What It Does

Provides a standardized APNs (iOS) and FCM v1 (Android) push notification pipeline:

- App code inserts a row into `np_push_outbox` (or Hasura triggers do it automatically on events)
- A Hasura event trigger fires a POST to `/push/dispatch`
- The push service dispatches to Apple or Google, tracks delivery state, and retries with exponential backoff
- Final status (delivered / failed) is written back to the outbox row

No per-app APNs/FCM bridge needed. One plugin handles all apps on the stack.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `PUSH_APNS_TEAM_ID` | | Apple Developer Team ID (10-char string) |
| `PUSH_APNS_KEY_ID` | | APNs Auth Key ID (10-char string from developer.apple.com) |
| `PUSH_APNS_KEY_PEM` | | APNs Auth Key content (raw PEM or file path , see Credentials) |
| `PUSH_APNS_BUNDLE_ID` | | App Bundle ID (e.g. `com.example.myapp`) |
| `PUSH_APNS_SANDBOX` | `0` | Set to `1` for APNs sandbox (development) |
| `PUSH_FCM_PROJECT_ID` | | Firebase project ID |
| `PUSH_FCM_SERVICE_ACCOUNT_JSON` | | FCM service account JSON (raw JSON or file path , see Credentials) |
| `PUSH_RETRY_MAX_ATTEMPTS` | `3` | Maximum delivery attempts before marking a message failed |
| `PUSH_RETRY_BACKOFF_BASE_MS` | `500` | Base backoff in ms (doubles each retry, capped at 30s) |

## Credential Setup

### APNs (iOS)

1. Go to [developer.apple.com](https://developer.apple.com) → Certificates, IDs & Profiles → Keys
2. Create an APNs Auth Key (.p8 file)
3. Note your Team ID, Key ID, and Bundle ID
4. Set the env vars in `.env.secrets` (never `.env.dev`):

```bash
PUSH_APNS_TEAM_ID=ABCDE12345
PUSH_APNS_KEY_ID=XYZ9876543
PUSH_APNS_BUNDLE_ID=com.example.myapp
# Preferred: paste PEM content directly (single-line with \n for newlines)
PUSH_APNS_KEY_PEM=-----BEGIN PRIVATE KEY-----\nMIGH...==\n-----END PRIVATE KEY-----
# Alternative: path to .p8 file mounted into container
# PUSH_APNS_KEY_PEM=/run/secrets/apns-key.p8
```

### FCM (Android)

1. Go to [Firebase Console](https://console.firebase.google.com) → Project Settings → Service Accounts
2. Generate a new private key (downloads a JSON file)
3. Set the env var in `.env.secrets`:

```bash
PUSH_FCM_PROJECT_ID=my-firebase-project-12345
# Preferred: paste JSON content directly
PUSH_FCM_SERVICE_ACCOUNT_JSON={"type":"service_account","project_id":"..."}
# Alternative: path to JSON file mounted into container
# PUSH_FCM_SERVICE_ACCOUNT_JSON=/run/secrets/fcm-sa.json
```

## Hasura Event Trigger Setup

Set up a Hasura event trigger that fires on INSERT to `np_push_outbox`:

```yaml
# Hasura metadata — add to your tables configuration
- table:
    name: np_push_outbox
    schema: public
  event_triggers:
    - name: push_dispatch
      definition:
        enable_manual: true
        insert:
          columns: "*"
      webhook: "http://push:8080/push/dispatch"
      headers:
        - name: Content-Type
          value: application/json
      retry_conf:
        num_retries: 0          # push plugin handles its own retry
        interval_sec: 10
        timeout_sec: 60
```

Or via Hasura Console: Tables → np_push_outbox → Events → Create Event Trigger.

## Sending a Push Notification

Insert a row into `np_push_outbox` from your application:

```sql
-- GraphQL mutation
mutation SendPush {
  insert_np_push_outbox_one(object: {
    device_token: "abc123...",
    platform: "ios",
    payload: {
      "aps": {
        "alert": {
          "title": "New message",
          "body": "You have a new message from Alice"
        },
        "badge": 1,
        "sound": "default"
      }
    }
  }) {
    id
    status
  }
}
```

For Android (FCM):

```graphql
mutation SendAndroidPush {
  insert_np_push_outbox_one(object: {
    device_token: "fcm-registration-token...",
    platform: "android",
    payload: {
      "notification": {
        "title": "New message",
        "body": "You have a new message from Alice"
      },
      "data": {
        "thread_id": "thread-123"
      }
    }
  }) {
    id
    status
  }
}
```

## Device Token Registration

Register device tokens when a user grants push permission in your app:

```bash
curl -X POST http://push:8080/push/devices \
  -H "Content-Type: application/json" \
  -d '{
    "device_token": "abc123...",
    "platform": "ios",
    "app_id": "myapp",
    "user_id": "user-uuid-here"
  }'
```

## Delivery States

Each outbox row progresses through these states:

| Status | Meaning |
|--------|---------|
| `pending` | Row inserted, waiting for Hasura event trigger to fire |
| `queued` | Dispatch accepted, delivery in progress |
| `delivered` | Provider confirmed delivery |
| `retrying` | First attempt failed, scheduled for retry |
| `failed` | All attempts exhausted; `last_error` has the provider error |

Query delivery status:

```graphql
query CheckDelivery($id: uuid!) {
  np_push_outbox_by_pk(id: $id) {
    status
    attempts
    last_error
    updated_at
  }
}
```

## Credential Rotation

APNs keys and FCM service accounts rotate on your security schedule. To rotate:

1. Generate the new key/service account from Apple / Firebase
2. Update `PUSH_APNS_KEY_PEM` or `PUSH_FCM_SERVICE_ACCOUNT_JSON` in `.env.secrets`
3. Restart the push container: `nself restart push`

The plugin reloads credentials from env on every startup. No cache to flush.

**Expired APNs key:** the plugin logs a clear error (`apns: credential error, ExpiredProviderToken`) and marks affected outbox rows as `failed`. No silent delivery loss.

## Security Notes

- Device tokens are stored in plain text in `np_push_devices`. They are semi-public identifiers, not secrets. The real secrets (APNs PEM, FCM JSON) are in env vars only, never in the database.
- FCM service account JSON is never logged. APNs JWT signing uses the loaded EC key without exposing it in any log line.
- The `/push/dispatch` endpoint validates that `device_token` is not a URL (SSRF guard). Tokens that look like `http://...` are rejected and the outbox row is marked failed.
- APNs uses JWT signing (ES256) with a fresh token per request, no stale-token risk.

## Relationship to notify Plugin

The `notify` plugin handles email (SMTP) and webhook delivery. The `push` plugin is its sibling for mobile push delivery. They are independent services that share the same `np_notify_outbox`-style pattern but operate on different tables and endpoints.

See: [[plugin-notify]] for email and webhook notification delivery.

---

[[Home]] | [[Plugin-Overview]] | [[plugin-notify]]

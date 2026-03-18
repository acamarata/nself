# Setting Up nself-mux

nself-mux is a Pro plugin that processes incoming emails and webhooks through a rule engine. Rules match incoming events and trigger actions — sending notifications, routing messages, classifying content, or calling other services.

---

## Prerequisites

- nself v0.9.9+ installed
- Pro or Max license key
- SMTP or IMAP credentials for the inbox you want to process
- Optional: nself-ai running (required for `ai_classify` and `use_claw_classify` actions)
- Optional: nself-voice running (required for `VoiceCall` and `VoiceTts` actions)

---

## Quick Start

Install the plugin:

```bash
nself plugin install mux
```

Register as a custom service:

```bash
CS_3=mux:express-ts:3102
CS_3_ROUTE=mux
CS_3_PUBLIC=false
CS_3_HEALTHCHECK=/health
CS_3_REPLICAS=1
CS_3_MEMORY=512M
CS_3_CPU=0.5
```

Copy service files and start:

```bash
cp -r ~/.nself/plugins/mux/ts/ services/mux/
nself build
docker compose up -d mux
```

---

## Required Environment Variables

```bash
# IMAP credentials for polling the inbox
PLUGIN_MUX_IMAP_HOST=mail.example.com
PLUGIN_MUX_IMAP_PORT=993
PLUGIN_MUX_IMAP_USER=inbox@example.com
PLUGIN_MUX_IMAP_PASS=yourpassword

# Token for authenticated calls to nself-ai (required if using ai_classify or companion actions)
PLUGIN_MUX_AI_TOKEN=nself_ai_tok_mux_xxxxxxxxxxxxx

# Shared secret for mux→claw classify RPC (required if using use_claw_classify: true)
MUX_CLAW_SHARED_SECRET=your-shared-secret-here
```

Create the AI token first:

```bash
nself ai token create --namespace mux --rate-limit 2000/hour
```

---

## Rule Structure

Rules are YAML files placed in `services/mux/rules/`. Each file is one rule. mux loads all rules at startup and watches for changes.

Basic rule structure:

```yaml
name: support-emails
description: Route support emails to the support queue
match:
  to: support@example.com
  subject_contains: ["help", "issue", "problem"]
actions:
  - type: forward
    to: team@example.com
  - type: tag
    tags: ["support", "needs-reply"]
priority: 10
enabled: true
```

Rules run in priority order (lower number = higher priority). The first matching rule wins unless `continue: true` is set.

---

## Actions Reference

### Standard Actions

#### forward

Forward the email to another address:

```yaml
- type: forward
  to: team@example.com
```

#### tag

Add tags to the message:

```yaml
- type: tag
  tags: ["sales", "hot-lead"]
```

#### webhook

POST the message payload to an external URL:

```yaml
- type: webhook
  url: https://your-api.example.com/hooks/email
  headers:
    Authorization: Bearer ${WEBHOOK_TOKEN}
```

> **Cloudflare SSL note:** Cloudflare's free plan SSL covers `example.com` and `*.example.com` only — not second-level subdomains like `webhooks.nclaw.example.com`. If your webhook URL uses a second-level subdomain behind Cloudflare DNS, either use a root-level subdomain (`webhooks.example.com`) or set the DNS record to DNS-only (grey cloud) so the origin certificate handles SSL directly. Cloudflare Pro and above support second-level subdomain SSL via Advanced Certificate Manager.

#### ai_classify

Send the email body to nself-ai for classification:

```yaml
- type: ai_classify
  task_class: Classify
  output_field: category
  labels: ["billing", "technical", "feedback", "spam"]
```

The classification result is stored in `message.meta.category` and available to subsequent actions in the same rule chain.

---

### New in v2: Extended Actions

#### CompanionNotify

Sends a card notification to the nClaw companion app. Use this to surface important events in the companion interface without sending an email or SMS.

```yaml
- type: companion_notify
  card_type: info        # info | warning | alert
  title: New support ticket
  body: "{{message.subject}} from {{message.from}}"
  priority: normal       # low | normal | high
```

`card_type` controls the visual style in the companion:
- `info` — blue, informational
- `warning` — amber, something to check
- `alert` — red, needs immediate attention

`priority` controls where the card appears in the feed:
- `low` — at the bottom, no badge
- `normal` — standard position
- `high` — pinned at top, triggers badge

Template variables `{{message.subject}}` and `{{message.from}}` are replaced with values from the incoming message.

Full example:

```yaml
name: alert-on-payment-failure
match:
  from_domain: stripe.com
  subject_contains: ["payment failed", "charge failed"]
actions:
  - type: companion_notify
    card_type: alert
    title: Payment failed
    body: "Stripe: {{message.subject}}"
    priority: high
```

#### VoiceCall

Initiates a phone call via Twilio. Requires nself-voice running and Twilio configured.

```yaml
- type: voice_call
  to: "+15551234567"
  message: "Alert: a critical error was detected on your server."
```

The `to` field must be an E.164 phone number. The `message` field is read aloud using the configured TTS voice.

Template variables work here too:

```yaml
- type: voice_call
  to: "+15551234567"
  message: "You have a new message from {{message.from}}."
```

Full example — call on-call engineer when a server alert arrives:

```yaml
name: call-oncall-on-critical-alert
match:
  subject_contains: ["CRITICAL", "DOWN", "OUTAGE"]
  from: alerts@monitoring.example.com
actions:
  - type: voice_call
    to: "+15559876543"
    message: "Critical alert: {{message.subject}}. Check your monitoring dashboard."
  - type: companion_notify
    card_type: alert
    title: Critical alert
    body: "{{message.subject}}"
    priority: high
priority: 1
```

#### VoiceTts

Plays a TTS message without making a phone call. Useful for local audio output (e.g., a speaker attached to a home server or kiosk).

```yaml
- type: voice_tts
  text: "New order received. Order ID {{message.meta.order_id}}."
  voice_id: en_US-lessac-medium   # optional, uses default if omitted
```

`voice_id` maps to a Piper model name or ElevenLabs voice ID.

Full example:

```yaml
name: announce-new-orders
match:
  from: orders@shopify.com
  subject_contains: ["You have a new order"]
actions:
  - type: voice_tts
    text: "New order from {{message.from}}."
```

---

## AI Classification with claw Classifier

The `ai_classify` action normally routes to the ai plugin directly. In some cases, you may want to use claw's classifier instead — for example, when claw has been fine-tuned on your domain or you want the classification to be aware of conversation history.

Add `use_claw_classify: true` to the action:

```yaml
- type: ai_classify
  task_class: Classify
  output_field: intent
  labels: ["sales-inquiry", "support-request", "spam", "newsletter"]
  use_claw_classify: true
```

When this flag is set, mux routes the classification call to claw's internal RPC endpoint (`POST /internal/classify`) instead of the ai plugin's direct classification endpoint. This requires nself-claw to be running and `MUX_CLAW_SHARED_SECRET` set in both services.

The `/internal/classify` endpoint requires a `Bearer` auth header using `MUX_CLAW_SHARED_SECRET`. The request body is:

```json
{
  "body": "<email body text>",
  "context": "<rule context — typically the rule name>",
  "labels": ["label1", "label2", "..."]
}
```

mux includes a circuit breaker: after 3 consecutive failures, it opens for 30 seconds and falls back to the standard ai plugin for classification. The circuit breaker resets automatically once claw becomes reachable again.

The output is the same regardless of which classifier runs. Only the routing changes.

---

## Importing Caller Tokens

mux needs `PLUGIN_MUX_AI_TOKEN` to make authenticated calls to nself-ai. This token must be created via the ai plugin CLI and stored in `.env`.

```bash
# Create the token
nself ai token create --namespace mux --rate-limit 2000/hour
# Output: nself_ai_tok_mux_xxxxxxxxxxxxxxxxxxxxxxxx

# Add to .env
PLUGIN_MUX_AI_TOKEN=nself_ai_tok_mux_xxxxxxxxxxxxxxxxxxxxxxxx
```

After adding the token, restart mux:

```bash
docker compose up -d --force-recreate mux
```

---

## Troubleshooting

### Rules are not matching

Check the rule file loaded without errors:

```bash
docker logs <mux_container> | grep "rule"
```

Invalid YAML causes the rule to be skipped silently. Validate your YAML:

```bash
python3 -c "import yaml, sys; yaml.safe_load(open('services/mux/rules/your-rule.yaml'))"
```

### CompanionNotify cards not appearing

The companion app must be connected to your backend. Open nClaw and confirm it shows the correct backend URL. Then verify the voice plugin is running (CompanionNotify routes through the voice plugin's push channel):

```bash
curl -s http://127.0.0.1:3103/health
```

### VoiceCall fails with "Twilio credentials missing"

`PLUGIN_MUX_AI_TOKEN` alone is not enough for VoiceCall. Voice requires the Twilio env vars in the voice service. Confirm they are set:

```bash
docker exec <voice_container> env | grep TWILIO
```

### "AI plugin not reachable" when using ai_classify

mux cannot reach nself-ai. Confirm ai is running and the token is correct:

```bash
curl -s http://127.0.0.1:3101/health
docker exec <mux_container> env | grep MUX_AI_TOKEN
```

---

## Related

- [nself-ai Setup](./ai-setup.md)
- [nself-claw Setup](./claw-setup.md)
- [nself-voice Setup](./voice-setup.md)
- [Pro Plugin Setup](./pro-plugin-setup.md)
- [Custom Services Reference](../configuration/custom-services.md)

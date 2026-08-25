# nself webhooks

**This command moved to a plugin.**

`nself webhooks` is no longer part of the CLI core. It inspects the durable
outbox the webhooks delivery service writes to, so it now ships with that
service rather than separately.

## Install

```bash
nself install webhooks
```

The webhooks plugin provides both the delivery service and this command.

## Commands

```bash
nself webhooks outbox status
nself webhooks outbox status --format json
```

`NSELF_WEBHOOK_OUTBOX_DIR` overrides the outbox location, which defaults to
`/var/lib/nself/webhook-outbox`.

---

← [[Commands]] · [[Plugin-Overview]] · [[Home]]

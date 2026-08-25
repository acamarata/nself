# nself infra

**This command moved to a plugin.**

`nself infra` is no longer part of the CLI core. Terraform-based cloud
provisioning is not part of the self-hosted backend lifecycle the core covers,
so it ships as a plugin for the people who want it.

## Install

```bash
nself install infra
```

Once installed, `nself infra ...` works exactly as before — the CLI proxies the
command to the plugin, and every flag, default and message is unchanged.

## Commands

```bash
nself infra plan    --provider hetzner --domain myapp.com
nself infra apply   --provider hetzner --domain myapp.com --force
nself infra destroy --provider hetzner --auto-approve
```

Providers: `aws`, `gcp`, `azure`, `hetzner`, `do`, `linode`. Terraform must be
on your `PATH`.

`nself infra apply` is gated and requires `--force`, which is how it behaved in
the CLI.

---

← [[Commands]] · [[Plugin-Overview]] · [[Home]]

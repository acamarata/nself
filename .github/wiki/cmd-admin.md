# nself admin

> Open the nSelf Admin UI in your browser.

## Synopsis

```
nself admin
```

## Description

`nself admin` opens the local nSelf Admin dashboard at `http://localhost:3021` in your default browser. The Admin UI is a local-only web interface that provides a GUI companion to the CLI — it runs as a Docker container on your machine and is never exposed to the internet.

The Admin UI must be running before you open it. Enable it as an optional service with `nself service enable admin`, then rebuild and restart your stack. Once running, `nself admin` is a shortcut to open the dashboard without typing the URL manually.

> **Note:** The Admin UI runs at `localhost:3021` on your own machine. It is distributed as a Docker image (`nself/nself-admin`) and launched via the nSelf stack — it is not a hosted service and is not accessible from outside your local network.

## Examples

```bash
# Open the Admin UI in your browser
nself admin

# Enable the Admin UI (first time setup)
nself service enable admin
nself build
nself restart

# Then open it
nself admin
```

← [[Commands]] | [[Home]] →

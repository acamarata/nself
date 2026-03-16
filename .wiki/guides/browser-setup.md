# Setting Up nself-browser

nself-browser adds headless browser automation to your backend. It handles screenshots, page scraping, PDF generation, and JavaScript execution. It uses Playwright under the hood, which requires Docker.

---

## Prerequisites

- nself v0.9.9+ installed
- Max license key (Max or Enterprise tier)
- Docker running on the host (required — Playwright runs inside a Docker container)

Confirm Docker is available:

```bash
docker info 2>/dev/null | head -2
# Should show server version
```

---

## Quick Start

Install the plugin:

```bash
nself plugin install browser
```

Register as a custom service:

```bash
CS_5=browser:express-ts:3104
CS_5_ROUTE=browser
CS_5_PUBLIC=false
CS_5_HEALTHCHECK=/health
CS_5_REPLICAS=1
CS_5_MEMORY=2G
CS_5_CPU=1.0
```

Copy service files and start:

```bash
cp -r ~/.nself/plugins/browser/ts/ services/browser/
nself build
docker compose up -d browser
```

Take your first screenshot:

```bash
curl -X POST http://127.0.0.1:3104/browser/screenshot \
  -H 'Authorization: Bearer <caller_token>' \
  -H 'Content-Type: application/json' \
  -d '{"url": "https://nself.org"}' \
  --output screenshot.png

# Opens screenshot.png
open screenshot.png
```

---

## URL Security Policy

By default, nself-browser blocks requests to private and internal addresses. This prevents server-side request forgery (SSRF) attacks where a malicious input tricks the browser into accessing your internal network.

Blocked by default:
- `localhost` and `127.0.0.1`
- All RFC-1918 ranges: `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`
- Link-local: `169.254.0.0/16`
- Loopback: `::1` (IPv6)

Blocked requests return `{"error": "URL not allowed — private or reserved address"}`.

### Adding to the Allowlist

To allow specific internal hosts (for example, to screenshot an internal dashboard):

```bash
# In .env
PLUGIN_BROWSER_ALLOWED_HOSTS=internal-dashboard.corp.example.com,reports.internal
```

Multiple hostnames are comma-separated. IP ranges are not supported in the allowlist — use hostnames.

Restart the browser service after updating:

```bash
docker compose up -d --force-recreate browser
```

### Disabling the Policy

For development only, you can disable URL filtering:

```bash
# In .env — NEVER in production
PLUGIN_BROWSER_DISABLE_URL_FILTER=true
```

---

## Stealth Mode

Standard Playwright installs are detectable by anti-bot systems. Stealth mode applies a set of patches that make the browser harder to fingerprint.

Enable it:

```bash
# In .env
PLUGIN_BROWSER_STEALTH=true
```

Stealth mode patches:
- Removes the `webdriver` property from `navigator`
- Randomizes canvas fingerprint
- Spoofs plugin and MIME type lists
- Normalizes Chrome runtime objects

Stealth mode is enabled per-pool. All browser instances in the pool use stealth when the flag is set. You cannot mix stealth and non-stealth in the same service instance.

---

## claw Integration

When nself-claw is running, the browser plugin registers itself as a `BrowserResearch` tool. claw can then browse the web as part of answering user questions.

No extra configuration required if both services are running. claw discovers the browser plugin via `PLUGIN_CLAW_BROWSER_TOKEN`.

```bash
# In .env — required for claw integration
PLUGIN_CLAW_BROWSER_TOKEN=nself_ai_tok_browser_xxxxx
```

Example: a user asks claw "what are the latest nself release notes?" claw calls `BrowserResearch` with `https://nself.org/releases`, extracts the text, and incorporates it into its answer.

claw passes the URL through the same security policy as direct API calls. Private URLs are blocked even when called through claw.

---

## API Reference

All endpoints require `Authorization: Bearer <caller_token>`.

### POST /browser/screenshot

Capture a full-page screenshot.

```json
{
  "url": "https://example.com",
  "width": 1280,
  "height": 720,
  "full_page": true,
  "format": "png"
}
```

Returns: PNG or JPEG binary, or a base64-encoded string if `return_base64: true`.

### POST /browser/scrape

Extract text and structured content from a page.

```json
{
  "url": "https://example.com/article",
  "extract": ["title", "text", "links", "images"],
  "wait_for": ".article-body"
}
```

`wait_for` accepts a CSS selector. The scraper waits for the element before extracting.

### POST /browser/pdf

Generate a PDF of a page.

```json
{
  "url": "https://example.com/report",
  "format": "A4",
  "margin": {"top": "1cm", "bottom": "1cm"}
}
```

Returns: PDF binary.

### POST /browser/execute

Run arbitrary JavaScript on a page.

```json
{
  "url": "https://example.com",
  "script": "return document.title",
  "wait_for_network": true
}
```

Returns: the value returned by the script.

---

## Pool Size Tuning

nself-browser keeps a pool of browser instances ready for requests. Each instance uses approximately 200–400MB of RAM.

```bash
# In .env
PLUGIN_BROWSER_POOL_SIZE=3
```

Default pool size is 3. Each instance is a separate Chromium process.

Memory estimate: `POOL_SIZE * 350MB + 200MB overhead`

For a VPS with 4GB RAM running the full nself stack, keep pool size at 2–3. On 8GB RAM, 5 is safe.

High pool sizes improve throughput for concurrent requests but do not speed up individual requests. If your workload is mostly sequential, the default of 3 is sufficient.

After changing pool size, restart the browser service:

```bash
docker compose up -d --force-recreate browser
```

---

## Troubleshooting

### "Docker is not available"

Playwright requires Docker. Confirm Docker is running:

```bash
docker ps
```

If Docker is not running, start it:

```bash
# Linux
sudo systemctl start docker
```

### Screenshot returns blank page

The page may require JavaScript to render. Try adding a wait:

```json
{
  "url": "https://example.com",
  "wait_for_network": true,
  "wait_ms": 2000
}
```

`wait_for_network: true` waits for all network requests to settle. `wait_ms` adds an additional fixed delay in milliseconds.

### "URL not allowed" for a public domain

The domain may resolve to a private IP (e.g., a split-horizon DNS setup). Check what the domain resolves to from the server:

```bash
dig +short yourdomain.com
```

If it resolves to an RFC-1918 IP, add it to `PLUGIN_BROWSER_ALLOWED_HOSTS`.

### "Pool exhausted — all browsers busy"

All browser instances are handling requests. Options:
1. Increase `PLUGIN_BROWSER_POOL_SIZE` (if RAM allows)
2. Add a retry with backoff in your calling code
3. Use the `queue: true` parameter to queue the request instead of getting an immediate error

```json
{
  "url": "https://example.com",
  "queue": true,
  "queue_timeout_ms": 10000
}
```

### High memory usage

Each Chromium instance is ~300–400MB. For a pool of 3, expect 1–1.2GB for the browser service alone.

Reduce pool size if memory is tight:

```bash
PLUGIN_BROWSER_POOL_SIZE=2
```

---

## Related

- [nself-claw Setup](./claw-setup.md)
- [nself-ai Setup](./ai-setup.md)
- [Custom Services Reference](../configuration/custom-services.md)
- [Pro Plugin Setup](./pro-plugin-setup.md)

#!/bin/sh
# nself-lb — LB helper installed on the load-balancer server by nself deploy.
#
# Usage:
#   nself-lb drain  <app-name>   # Remove app from upstream pool (weight 0)
#   nself-lb enable <app-name>   # Re-add app to upstream pool (weight 1)
#
# The helper is idempotent: running drain on an already-drained backend or
# enable on an already-enabled backend is safe and exits 0.
#
# Supported backends (auto-detected):
#   HAProxy — rewrites the weight via the stats socket (socat)
#   nginx   — rewrites the upstream weight in the config fragment and reloads
#
# Exit codes:
#   0  success
#   1  usage error or backend manipulation failure
#   2  unsupported / unconfigured backend

set -eu

PROG="nself-lb"
HAPROXY_SOCKET="${HAPROXY_SOCKET:-/var/run/haproxy/admin.sock}"
NGINX_CONF_DIR="${NGINX_CONF_DIR:-/etc/nginx/conf.d}"

usage() {
    printf "usage: %s drain|enable <app-name>\n" "$PROG" >&2
    exit 1
}

# ---- argument validation ------------------------------------------------

if [ $# -ne 2 ]; then
    usage
fi

ACTION="$1"
APP="$2"

case "$ACTION" in
    drain|enable) ;;
    *) usage ;;
esac

if [ -z "$APP" ]; then
    printf "%s: app-name must not be empty\n" "$PROG" >&2
    exit 1
fi

# ---- backend auto-detection --------------------------------------------

BACKEND=""

if command -v socat >/dev/null 2>&1 && [ -S "$HAPROXY_SOCKET" ]; then
    BACKEND="haproxy"
elif command -v nginx >/dev/null 2>&1; then
    BACKEND="nginx"
fi

if [ -z "$BACKEND" ]; then
    printf "%s: no supported LB backend found (tried HAProxy via socat, nginx)\n" "$PROG" >&2
    exit 2
fi

# ---- HAProxy backend ---------------------------------------------------

haproxy_drain() {
    # Set server weight to 0 (drained); HAProxy will finish in-flight requests
    # before stopping new ones. Idempotent.
    printf "set weight backend/%s 0\n" "$APP" | socat stdio "$HAPROXY_SOCKET" >/dev/null 2>&1 || {
        printf "%s: haproxy drain failed for %s\n" "$PROG" "$APP" >&2
        exit 1
    }
}

haproxy_enable() {
    # Restore server weight to 1 (active). Idempotent.
    printf "set weight backend/%s 1\n" "$APP" | socat stdio "$HAPROXY_SOCKET" >/dev/null 2>&1 || {
        printf "%s: haproxy enable failed for %s\n" "$PROG" "$APP" >&2
        exit 1
    }
}

# ---- nginx backend -----------------------------------------------------

# nginx fragment naming convention: $NGINX_CONF_DIR/upstream-<app>.conf
# Expected format:
#   upstream <app> {
#       server <host>:<port> weight=1;  # managed by nself-lb
#   }

nginx_set_weight() {
    _weight="$1"
    _conf="${NGINX_CONF_DIR}/upstream-${APP}.conf"

    if [ ! -f "$_conf" ]; then
        printf "%s: nginx upstream config not found: %s\n" "$PROG" "$_conf" >&2
        exit 1
    fi

    # Rewrite `weight=<N>` on the line containing `# managed by nself-lb`.
    # Use sed in-place (POSIX: write to tmp then move).
    _tmp="${_conf}.nself-lb.tmp"
    sed "s/weight=[0-9]*/weight=${_weight}/g" "$_conf" > "$_tmp" && mv "$_tmp" "$_conf"

    # Reload nginx to apply.
    nginx -s reload >/dev/null 2>&1 || {
        printf "%s: nginx reload failed\n" "$PROG" >&2
        exit 1
    }
}

nginx_drain()  { nginx_set_weight 0; }
nginx_enable() { nginx_set_weight 1; }

# ---- dispatch ----------------------------------------------------------

case "$BACKEND-$ACTION" in
    haproxy-drain)  haproxy_drain  ;;
    haproxy-enable) haproxy_enable ;;
    nginx-drain)    nginx_drain    ;;
    nginx-enable)   nginx_enable   ;;
    *)
        printf "%s: internal error: unknown backend-action pair %s-%s\n" "$PROG" "$BACKEND" "$ACTION" >&2
        exit 1
        ;;
esac

printf "%s: %s %s OK (backend=%s)\n" "$PROG" "$ACTION" "$APP" "$BACKEND"
exit 0

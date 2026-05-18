package trust

// dnsmasqConfLine is the dnsmasq configuration line used on both macOS and
// Linux to enable wildcard .local DNS resolution.
const dnsmasqConfLine = "address=/.local/127.0.0.1"

// resolverPath is the macOS system resolver configuration file path. Defined
// here (no build tag) so that tests on all platforms can reference the constant.
const resolverPath = "/etc/resolver/local"

// resolverContent is the content written to /etc/resolver/local on macOS to
// point the system resolver at the local dnsmasq instance. Defined here
// (no build tag) so that tests on all platforms can reference the constant.
const resolverContent = "nameserver 127.0.0.1\n"

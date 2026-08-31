package trust

// pfAnchorPath is the macOS pf anchor file path. Defined here (no build tag)
// so that tests on all platforms can reference the constant.
const pfAnchorPath = "/etc/pf.anchors/nself.local"

// launchDaemonPlistPath is the macOS LaunchDaemon plist path. Defined here
// (no build tag) so that tests on all platforms can reference the constant.
const launchDaemonPlistPath = "/Library/LaunchDaemons/com.nself.portforward.plist"

// launchDaemonPlistContent is the plist XML written to launchDaemonPlistPath.
// Defined here (no build tag) so that tests on all platforms can reference
// the constant.
const launchDaemonPlistContent = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.nself.portforward</string>
    <key>ProgramArguments</key>
    <array>
        <string>/sbin/pfctl</string>
        <string>-ef</string>
        <string>/etc/pf.anchors/nself.local</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
</dict>
</plist>
`

# nself maintenance

**Free disk space and schedule automatic daily cleanup.**

The `maintenance` command group provides tools for keeping your ɳSelf host
healthy over time. Use it to reclaim disk space consumed by stale Docker
images, old log archives, and journal entries, and to install a timer that
runs the same cleanup automatically every night.

---

## nself maintenance disk-cleanup

Run all three cleanup steps immediately and report before/after disk usage.

```bash
nself maintenance disk-cleanup
```

### What it does

| Step | Command | Effect |
|------|---------|--------|
| Docker prune | `docker system prune -af --volumes=false` | Removes stopped containers and unused images. Volumes are preserved. |
| Log rotation | `find /var/log -name '*.gz' -mtime +14 -delete` | Removes compressed log files older than 14 days. |
| Journal vacuum | `journalctl --vacuum-time=7d` | Removes journal entries older than 7 days. Linux only; no-op on macOS. |

### Example output

```
nSelf Maintenance — Disk cleanup

Before cleanup
  Used:  42.3 GB / 100.0 GB total  [############------------------] 42%
  Free: 57.7 GB

Running cleanup steps
  Docker prune: ...
  ...

After cleanup
  Used:  38.1 GB / 100.0 GB total  [###########-------------------] 38%
  Free: 61.9 GB

Freed 4.20 GB (disk 42% → 38% used)
```

---

## nself maintenance schedule

Install or remove the system-level daily cleanup timer.

```bash
nself maintenance schedule --daily   # install timer (requires sudo/root)
nself maintenance schedule --off     # remove timer  (requires sudo/root)
```

### Flags

| Flag | Description |
|------|-------------|
| `--daily` | Install a timer that runs `disk-cleanup` every day at 03:00 local time |
| `--off` | Remove the daily cleanup timer |

### Platform behaviour

| Platform | Mechanism |
|----------|-----------|
| Linux | Writes `/etc/systemd/system/nself-disk-cleanup.{service,timer}` and enables via `systemctl` |
| macOS | Writes `/Library/LaunchDaemons/org.nself.disk-cleanup.plist` and loads via `launchctl` |
| Windows | Not supported. Use Task Scheduler manually. |

Both mechanisms require root/sudo. Run with `sudo nself maintenance schedule --daily` if prompted.

---

## nself maintenance status

Show whether the daily timer is installed and the current disk usage.

```bash
nself maintenance status
```

### Example output

```
nSelf Maintenance — Scheduler status

  Platform : linux (systemd)
  Daily cleanup timer is NOT enabled
  Run `nself maintenance schedule --daily` to enable it.

Current disk usage
  Used:  42.3 GB / 100.0 GB total  [############------------------] 42%
  Free: 57.7 GB
```

---

## Integration with nself doctor

`nself doctor` and `nself doctor --deep` both check disk space. When disk
usage exceeds 70%, the check result includes:

```
warn  Disk space  57.7 GB free — disk is 72% full — run `nself maintenance schedule --daily` to enable automatic daily cleanup
```

Enable the daily timer to keep the warning from recurring.

---

## See also

- [[cmd-doctor.md]], full system diagnostics
- [[cmd-plugin-logs.md]], view plugin log output

[[Home]]

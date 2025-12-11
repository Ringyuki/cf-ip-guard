# cf-ip-guard

Keep local ipsets in sync with Cloudflare IP ranges and consume them in firewall rules.

## What it does
- Fetches Cloudflare IPv4/IPv6 ranges from `https://api.cloudflare.com/client/v4/ips`.
- Maintains two ipsets (default: `cloudflare4`, `cloudflare6`) with atomic swap.
- Does **not** create iptables/nftables rules; you must reference the ipsets yourself.

## Prerequisites
- Linux with `ipset` and `iptables` installed.
- Root privileges (ipset operations need `CAP_NET_ADMIN`).
- Go toolchain if building from source.
- Create your own firewall rules that use the ipsets (examples below).

## Build & run (manual)
```bash
go build -o cf-ip-guard ./...
sudo mv cf-ip-guard /usr/local/bin/
sudo cf-ip-guard daemon --ipset4 cloudflare4 --ipset6 cloudflare6 --interval 30m --log-level info
```

## Deploy with systemd
```bash
./deploy/install.sh
# optional: edit flags
sudo sed -n '1,20p' /etc/cf-ip-guard.env
sudo systemctl restart cf-ip-guard
```
- Unit file: `deploy/cf-ip-guard.service` (installs to `/etc/systemd/system/cf-ip-guard.service`).
- Extra CLI flags: set `CF_IP_GUARD_OPTS` in `/etc/cf-ip-guard.env` (e.g. `--interval 10m --log-level debug`).

## Firewall rule examples (iptables)
Make sure the ipsets exist (the daemon creates/syncs them). Add your own rules, for example:
```bash
# Allow inbound traffic only if source is in Cloudflare IPv4/IPv6 sets
iptables -I INPUT -m set --match-set cloudflare4 src -j ACCEPT
ip6tables -I INPUT -m set --match-set cloudflare6 src -j ACCEPT

# Optionally drop others for the same service/port in later rules (not shown)
```
Adjust chains/ports to your policy. If you already use nftables, create equivalent rules referencing the ipsets.

## Runtime notes
- Defaults: interval 30m, ipset names `cloudflare4`/`cloudflare6`, API URL Cloudflare `/ips`.
- On startup the daemon performs an immediate fetch/update, then loops on the interval.
- Logs go to stderr; configure level via `--log-level` or `CF_IP_GUARD_OPTS`.


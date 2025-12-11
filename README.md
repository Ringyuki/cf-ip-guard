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

## Firewall rule examples (iptables, only 80/443)
The design goal is to allow only Cloudflare IPs to reach HTTP/HTTPS. Ensure the ipsets exist (daemon creates/syncs them), then:
```bash
# Allow HTTP/HTTPS from Cloudflare IPv4
sudo iptables -I INPUT -p tcp -m set --match-set cloudflare4 src --dport 80  -j ACCEPT
sudo iptables -I INPUT -p tcp -m set --match-set cloudflare4 src --dport 443 -j ACCEPT

# Allow HTTP/HTTPS from Cloudflare IPv6
sudo ip6tables -I INPUT -p tcp -m set --match-set cloudflare6 src --dport 80  -j ACCEPT
sudo ip6tables -I INPUT -p tcp -m set --match-set cloudflare6 src --dport 443 -j ACCEPT

# Drop all other HTTP/HTTPS traffic
sudo iptables  -A INPUT -p tcp --dport 80  -j DROP
sudo iptables  -A INPUT -p tcp --dport 443 -j DROP
sudo ip6tables -A INPUT -p tcp --dport 80  -j DROP
sudo ip6tables -A INPUT -p tcp --dport 443 -j DROP
```
Adjust chains (e.g., use a dedicated service chain) and insertion order to fit your policy. For nftables, create equivalent rules referencing the same ipsets.

## Runtime notes
- Defaults: interval 30m, ipset names `cloudflare4`/`cloudflare6`, API URL Cloudflare `/ips`.
- On startup the daemon performs an immediate fetch/update, then loops on the interval.
- Logs go to stderr; configure level via `--log-level` or `CF_IP_GUARD_OPTS`.


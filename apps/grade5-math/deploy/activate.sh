#!/usr/bin/env bash
# Uploaded package only. This script never compiles code on the server.
set -euo pipefail
archive="$1"; checksum="$2"; release="$3"
[[ "$release" =~ ^[0-9]{8}T[0-9]{6}Z-[0-9a-f]{12}$ ]] || exit 2
[[ "$checksum" =~ ^[0-9a-f]{64}$ ]] || exit 2
[ "$(id -u)" = 0 ] || { printf 'Root required\n' >&2; exit 2; }
base=/opt/grade5-math
state=/var/lib/grade5-math
web=/var/www/grade5-math-current
conf=/etc/nginx/conf.d/start.deey.top.conf
unit=/etc/systemd/system/grade5-math.service
backup="/var/backups/grade5-math/$release"
mkdir -p "$base/releases" "$backup"
chmod 700 /var/backups/grade5-math "$backup"
exec 9>"$base/deploy.lock"
flock -n 9 || { printf 'Another deploy is running\n' >&2; exit 1; }
printf '%s  %s\n' "$checksum" "$archive" | sha256sum -c -
[ ! -e "$base/releases/$release" ] || { printf 'Release already exists\n' >&2; exit 1; }
# Preserve both existing sites and the default server. Sensitive raw backups stay root-only.
nginx -t
nginx -T > "$backup/nginx-before.txt" 2>&1
tar -czf "$backup/nginx-before.tgz" -C /etc nginx
sha256sum /etc/nginx/nginx.conf /etc/nginx/conf.d/mkd-app.idaay.com.conf /usr/share/nginx/html/index.html > "$backup/protected.sha256"
www_before="$(curl -fsS --max-time 10 --resolve www.idaay.com:80:127.0.0.1 http://www.idaay.com/ | sha256sum | awk '{print $1}')"
mkd_before="$(curl -fsS --max-time 10 --resolve mkd-app.idaay.com:443:127.0.0.1 https://mkd-app.idaay.com/ | sha256sum | awk '{print $1}')"
printf '%s\n%s\n' "$www_before" "$mkd_before" > "$backup/old-site-bodies.sha256"
previous=""
if [ -L "$base/current" ]; then previous="$(readlink -f "$base/current")"; elif [ -e "$base/current" ]; then printf "Unexpected current path\n" >&2; exit 1; fi
[ ! -f "$unit" ] || cp -p "$unit" "$backup/grade5-math.service"
[ ! -f "$conf" ] || cp -p "$conf" "$backup/start.deey.top.conf"
if [ -f "$state/math.db" ]; then sqlite3 "$state/math.db" ".backup '$backup/math.db'"; fi
if [ -e "$web" ] || [ -L "$web" ]; then [ "$(readlink "$web")" = "$base/current/web" ] || { printf 'Unexpected web path, refusing to replace\n' >&2; exit 1; }; fi
mkdir "$base/releases/$release"
tar -xzf "$archive" -C "$base/releases/$release"
cd "$base/releases/$release"
sha256sum -c SHA256SUMS
chmod 755 bin/math-server
./bin/math-server -check -content content
id grade5math >/dev/null 2>&1 || useradd --system --no-create-home --shell /sbin/nologin grade5math
install -d -o grade5math -g grade5math -m 700 "$state"
mkdir -p /var/www/grade5-math-acme/.well-known/acme-challenge
new_conf=0; switched=0
rollback() {
  status=$?
  trap - ERR
  printf 'Deployment failed; rolling back application only. Backup: %s\n' "$backup" >&2
  if [ "$switched" = 1 ]; then
    if [ -n "$previous" ]; then
      ln -sfn "$previous" "$base/current.next"; mv -Tf "$base/current.next" "$base/current"
      if [ -f "$backup/grade5-math.service" ]; then cp -p "$backup/grade5-math.service" "$unit"; fi
      systemctl daemon-reload; systemctl restart grade5-math || true
    else
      systemctl disable --now grade5-math || true
    fi
  fi
  if [ "$new_conf" = 1 ]; then mv "$conf" "$backup/start.deey.top.conf.failed"; nginx -t && systemctl reload nginx; fi
  exit "$status"
}
trap rollback ERR
install -m 644 ops/grade5-math.service "$unit"
ln -sfn "$base/releases/$release" "$base/current.next"
mv -Tf "$base/current.next" "$base/current"
[ -L "$web" ] || ln -s "$base/current/web" "$web"
switched=1
if [ ! -f "$conf" ]; then install -m 644 ops/start.deey.top.http.conf "$conf"; new_conf=1; fi
nginx -t
systemctl daemon-reload
systemctl enable grade5-math
systemctl restart grade5-math
healthy=0
for attempt in $(seq 1 20); do
  if curl -fsS --max-time 2 http://127.0.0.1:18081/api/health > "$backup/health.json"; then healthy=1; break; fi
  sleep 1
done
[ "$healthy" = 1 ]
if [ "$new_conf" = 1 ]; then systemctl reload nginx; fi
curl -fsS --max-time 5 --resolve start.deey.top:80:127.0.0.1 http://start.deey.top/api/health
sha256sum -c "$backup/protected.sha256"
www_after="$(curl -fsS --max-time 10 --resolve www.idaay.com:80:127.0.0.1 http://www.idaay.com/ | sha256sum | awk '{print $1}')"
mkd_after="$(curl -fsS --max-time 10 --resolve mkd-app.idaay.com:443:127.0.0.1 https://mkd-app.idaay.com/ | sha256sum | awk '{print $1}')"
[ "$www_before" = "$www_after" ] && [ "$mkd_before" = "$mkd_after" ]
systemctl is-active grade5-math mkd-app nginx
trap - ERR
printf 'DEPLOYED %s\nPREVIOUS %s\nBACKUP %s\n' "$release" "${previous:-none}" "$backup"

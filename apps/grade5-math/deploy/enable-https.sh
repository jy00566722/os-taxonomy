#!/usr/bin/env bash
# Run on the server after uploading the matching config and hook into /var/tmp.
set -euo pipefail
backup="/var/backups/grade5-math/https-$(date -u +%Y%m%dT%H%M%SZ)"
conf=/etc/nginx/conf.d/start.deey.top.conf
envfile=/etc/sysconfig/grade5-math
mkdir -p "$backup"
chmod 700 "$backup"
cp -p "$conf" "$backup/start.deey.top.conf"
if [ -f "$envfile" ]; then cp -p "$envfile" "$backup/environment"; fi
sha256sum /etc/nginx/nginx.conf /etc/nginx/conf.d/mkd-app.idaay.com.conf /usr/share/nginx/html/index.html /etc/letsencrypt/renewal/mkd-app.idaay.com.conf /etc/letsencrypt/live/mkd-app.idaay.com/fullchain.pem > "$backup/protected.sha256"
www_before="$(curl -fsS --resolve www.idaay.com:80:127.0.0.1 http://www.idaay.com/ | sha256sum | awk '{print $1}')"
mkd_before="$(curl -fsS --resolve mkd-app.idaay.com:443:127.0.0.1 https://mkd-app.idaay.com/ | sha256sum | awk '{print $1}')"
install -d -m 755 /opt/grade5-math/ops /var/www/grade5-math-acme/.well-known/acme-challenge
install -m 755 /var/tmp/grade5-math-certbot-deploy.sh /opt/grade5-math/ops/certbot-deploy.sh
certbot certonly --non-interactive --webroot -w /var/www/grade5-math-acme --cert-name start.deey.top -d start.deey.top --keep-until-expiring --deploy-hook /opt/grade5-math/ops/certbot-deploy.sh
rollback() {
  status=$?; trap - ERR
  cp -p "$backup/start.deey.top.conf" "$conf"
  if [ -f "$backup/environment" ]; then cp -p "$backup/environment" "$envfile"; else printf 'MATH_EXTRA_FLAGS=\n' > "$envfile"; fi
  systemctl restart grade5-math
  nginx -t && systemctl reload nginx
  printf 'HTTPS configuration rolled back. Backup: %s\n' "$backup" >&2
  exit "$status"
}
trap rollback ERR
install -m 644 /var/tmp/grade5-math-start.https.conf "$conf"
printf 'MATH_EXTRA_FLAGS=-secure-cookies\n' > "$envfile"
chmod 600 "$envfile"
nginx -t
systemctl restart grade5-math
systemctl reload nginx
ready=0
for attempt in $(seq 1 20); do
  if curl -fsS --max-time 3 --resolve start.deey.top:443:127.0.0.1 https://start.deey.top/api/health > "$backup/https-health.json" 2>/dev/null; then ready=1; break; fi
  sleep 1
done
[ "$ready" = 1 ]
sha256sum -c "$backup/protected.sha256"
[ "$www_before" = "$(curl -fsS --resolve www.idaay.com:80:127.0.0.1 http://www.idaay.com/ | sha256sum | awk '{print $1}')" ]
[ "$mkd_before" = "$(curl -fsS --resolve mkd-app.idaay.com:443:127.0.0.1 https://mkd-app.idaay.com/ | sha256sum | awk '{print $1}')" ]
systemctl is-active grade5-math mkd-app nginx
systemctl is-enabled certbot-renew.timer
trap - ERR
cat "$backup/https-health.json"
openssl x509 -in /etc/letsencrypt/live/start.deey.top/fullchain.pem -noout -issuer -dates -ext subjectAltName
printf 'HTTPS_BACKUP=%s\n' "$backup"

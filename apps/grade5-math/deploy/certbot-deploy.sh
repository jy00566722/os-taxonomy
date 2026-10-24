#!/usr/bin/env bash
set -euo pipefail
# Saved only in this certificate's renewal config; not a global renewal hook.
case " ${RENEWED_DOMAINS:-} " in
  *" start.deey.top "*) /usr/sbin/nginx -t; /usr/bin/systemctl reload nginx ;;
esac

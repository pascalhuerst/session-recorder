#!/bin/sh
set -e

ENABLED=/etc/nginx/sites-enabled/session-recorder-web
DEFAULT_AVAIL=/etc/nginx/sites-available/default
DEFAULT_ENABLED=/etc/nginx/sites-enabled/default

case "$1" in
  remove|deconfigure)
    if [ -L "$ENABLED" ]; then
      rm -f "$ENABLED"
      # Restore the stock default site we disabled on install so the host is
      # not left without a site on :80 (best effort).
      if [ -f "$DEFAULT_AVAIL" ] && [ ! -e "$DEFAULT_ENABLED" ]; then
        ln -s "$DEFAULT_AVAIL" "$DEFAULT_ENABLED"
      fi
      if [ -d /run/systemd/system ] && command -v nginx >/dev/null 2>&1; then
        nginx -t && systemctl reload nginx || true
      fi
    fi
    ;;
esac

exit 0

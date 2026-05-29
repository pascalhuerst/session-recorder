#!/bin/sh
set -e

# rpm %post: numeric arg (1 install, 2 upgrade). The site file is already in
# /etc/nginx/conf.d/session-recorder-web.conf, which the stock nginx.conf
# auto-includes. Validate the full config and reload; if it clashes — most
# commonly a duplicate `default_server` on :80 from Fedora's stock server block
# in /etc/nginx/nginx.conf — roll our file back so nginx is never left unable to
# start, and tell the user how to resolve it.

CONF=/etc/nginx/conf.d/session-recorder-web.conf

if [ "$1" -ge 1 ] 2>/dev/null; then
  if command -v nginx >/dev/null 2>&1; then
    if nginx -t 2>/dev/null; then
      if [ -d /run/systemd/system ]; then
        systemctl reload nginx 2>/dev/null || systemctl restart nginx || true
      fi
      echo "session-recorder-web enabled; nginx reloaded (serving on port 80)."
    else
      rm -f "$CONF"
      echo "WARNING: nginx config test failed — session-recorder-web left disabled." >&2
      echo "This is usually a duplicate 'default_server' on :80 from the stock" >&2
      echo "server block in /etc/nginx/nginx.conf. Remove that default_server (or" >&2
      echo "the whole stock server block), reinstall this package, then:" >&2
      echo "  nginx -t && systemctl reload nginx" >&2
    fi
  fi
  echo "Edit $CONF to change the port or the /grpc and /minio upstreams."
fi

exit 0

#!/bin/sh
set -e

# rpm %postun: numeric arg, 0 on final removal (1 during upgrade). rpm has
# already removed the conf.d file by this point; just reload nginx so it drops
# our server block. Only on final removal, and only if the config still tests OK.

if [ "$1" = "0" ]; then
  if [ -d /run/systemd/system ] && command -v nginx >/dev/null 2>&1; then
    nginx -t 2>/dev/null && systemctl reload nginx 2>/dev/null || true
  fi
fi

exit 0

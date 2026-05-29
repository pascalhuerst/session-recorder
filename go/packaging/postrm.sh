#!/bin/sh
set -e

# deb postrm: arg "remove"/"purge"; rpm %postun: numeric arg, 0 on final
# removal. Reload systemd after the unit file is gone.

if [ "$1" = "remove" ] || [ "$1" = "purge" ] || [ "$1" = "0" ]; then
  if [ -d /run/systemd/system ]; then
    systemctl daemon-reload || true
  fi
fi

# The dedicated user is intentionally left in place to avoid orphaning files it
# may own. Remove manually if desired:
#   userdel session-recorder

exit 0

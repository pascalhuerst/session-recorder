#!/bin/sh
set -e

# deb prerm: arg "remove"/"deconfigure"; rpm %preun: numeric arg, 0 on final
# removal (1 during an upgrade). Stop/disable the service only when it's going
# away for good.

if [ "$1" = "remove" ] || [ "$1" = "deconfigure" ] || [ "$1" = "0" ]; then
  if [ -d /run/systemd/system ]; then
    systemctl stop session-recorder-server.service || true
    systemctl disable session-recorder-server.service || true
  fi
fi

exit 0

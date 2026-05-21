#!/bin/sh
set -e

case "$1" in
  remove|deconfigure)
    if [ -d /run/systemd/system ]; then
      systemctl stop session-recorder-server.service || true
      systemctl disable session-recorder-server.service || true
    fi
    ;;
esac

exit 0

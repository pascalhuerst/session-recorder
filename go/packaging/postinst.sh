#!/bin/sh
set -e

USER=session-recorder
GROUP=session-recorder
STATE_DIR=/var/lib/session-recorder-server

case "$1" in
  configure)
    if ! getent group "$GROUP" >/dev/null 2>&1; then
      addgroup --system "$GROUP"
    fi
    if ! getent passwd "$USER" >/dev/null 2>&1; then
      adduser --system --no-create-home --ingroup "$GROUP" \
        --home "$STATE_DIR" --shell /usr/sbin/nologin "$USER"
    fi
    if [ -d /run/systemd/system ]; then
      systemctl daemon-reload || true
    fi
    echo "session-recorder-server installed. Edit /etc/default/session-recorder-server,"
    echo "then: systemctl enable --now session-recorder-server"
    ;;
esac

exit 0

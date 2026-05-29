#!/bin/sh
set -e

# Runs as the deb postinst (arg "configure") and the rpm %post scriptlet
# (numeric arg: 1 on install, 2 on upgrade). Uses low-level useradd/groupadd so
# it works on both Debian and Fedora (no Debian-only adduser).

USER=session-recorder
GROUP=session-recorder
STATE_DIR=/var/lib/session-recorder-server

if [ "$1" = "configure" ] || { [ "$1" -ge 1 ] 2>/dev/null; }; then
  if ! getent group "$GROUP" >/dev/null 2>&1; then
    groupadd --system "$GROUP"
  fi
  if ! getent passwd "$USER" >/dev/null 2>&1; then
    useradd --system --gid "$GROUP" --no-create-home \
      --home-dir "$STATE_DIR" --shell /usr/sbin/nologin "$USER"
  fi
  if [ -d /run/systemd/system ]; then
    systemctl daemon-reload || true
  fi
  echo "session-recorder-server installed. Edit /etc/default/session-recorder-server,"
  echo "then: systemctl enable --now session-recorder-server"
fi

exit 0

#!/bin/sh
set -e

SITE=session-recorder-web
AVAIL=/etc/nginx/sites-available/$SITE
ENABLED=/etc/nginx/sites-enabled/$SITE
DEFAULT_ENABLED=/etc/nginx/sites-enabled/default

case "$1" in
  configure)
    if [ ! -d /etc/nginx/sites-enabled ]; then
      echo "No /etc/nginx/sites-enabled directory; enable the site manually"
      echo "(see the header of $AVAIL)."
      exit 0
    fi

    # Our site is a default_server, so the stock nginx default site (also a
    # default_server on :80) must be disabled or nginx -t fails with a
    # duplicate. Only the sites-enabled symlink is removed; the file in
    # sites-available stays, so it can be re-enabled later.
    if [ -L "$DEFAULT_ENABLED" ]; then
      rm -f "$DEFAULT_ENABLED"
      echo "Disabled stock nginx default site (sites-enabled/default)."
    fi

    [ -e "$ENABLED" ] || ln -s "$AVAIL" "$ENABLED"

    # Validate before reloading. If the config is bad, roll back our symlink so
    # nginx is never left in a state where it can't start.
    if command -v nginx >/dev/null 2>&1; then
      if nginx -t 2>/dev/null; then
        if [ -d /run/systemd/system ]; then
          systemctl reload nginx 2>/dev/null || systemctl restart nginx || true
        fi
        echo "session-recorder-web enabled; nginx reloaded (serving on port 80)."
      else
        rm -f "$ENABLED"
        echo "WARNING: nginx config test failed — leaving session-recorder-web disabled." >&2
        echo "Fix $AVAIL, then re-enable with:" >&2
        echo "  ln -s $AVAIL $ENABLED && nginx -t && systemctl reload nginx" >&2
      fi
    fi

    echo "Edit $AVAIL to change the port or the /grpc and /minio upstreams."
    ;;
esac

exit 0

#!/bin/sh
# Run only explicit CI fixtures/installed-app qualification in private state.
# The caller supplies an unprivileged account and HOME/XDG paths. No application
# or sandbox flags are introduced here, and the checkout is not the working dir.
set -eu
if [ "$(id -u)" -eq 0 ]; then
  echo 'Linux qualification must run as an unprivileged user' >&2
  exit 1
fi
cd "$HOME"
exec dbus-run-session -- xvfb-run -a -s '-screen 0 1280x900x24 -nolisten tcp -noreset' "$@"

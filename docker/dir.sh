#!/bin/bash

set -euo pipefail

export PATH=$PATH:/usr/bin/core_perl
mkdir -p /buffer/$AURORA_PACKAGE

rm /var/lib/pacman/db.lck 2> /dev/null || true

sudo -u nobody mkdir /app/build/$AURORA_PACKAGE

cd /app/build/$AURORA_PACKAGE

if [[ ! "${AURORA_CLONE_URL:-}" ]]; then
    AURORA_CLONE_URL=https://aur.archlinux.org/$AURORA_PACKAGE.git
fi

sudo -u nobody git clone "${AURORA_CLONE_URL}" .

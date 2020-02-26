#!/bin/bash

set -euo pipefail

export PATH=$PATH:/usr/bin/core_perl

buildtime=$(date +%s)
pkg=${AURORA_PACKAGE}

rm /var/lib/pacman/db.lck 2> /dev/null \
        || true

sudo -u nobody mkdir /app/build/$pkg

cd /app/build/$pkg

if [[ ! "${AURORA_CLONE_URL:-}" ]]; then
    AURORA_CLONE_URL=https://aur.archlinux.org/$pkg.git
fi
sudo -u nobody git clone "${AURORA_CLONE_URL}" .

sudo -u nobody -E makepkg --syncdeps --noconfirm

mkdir -p /buffer/$pkg
find ./ -maxdepth 1 -type f -name '*.pkg.*' -printf '%P\n' | while read filename; do
    cp "${filename}" "/buffer/$pkg/${buildtime}.${filename}"
done

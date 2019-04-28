#!/bin/bash

set -euo pipefail

export PATH=$PATH:/usr/bin/core_perl

buildtime=$(date +%s)
pkg=${AURORA_PACKAGE}

rm /var/lib/pacman/db.lck 2> /dev/null \
        || true

sudo -u nobody mkdir /app/build/$pkg

cd /app/build/$pkg
sudo -u nobody git clone https://aur.archlinux.org/$pkg.git .

sudo -u nobody -E makepkg --syncdeps --noconfirm

mkdir /buffer/$pkg
find ./ -maxdepth 1 -type f -name '*.pkg.*' | while read filename; do
    cp "${filename}" "/buffer/$pkg/${buildtime}.${filename}"
done

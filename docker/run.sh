#!/bin/bash

set -euo pipefail

export PATH=$PATH:/usr/bin/core_perl

pkg=${AURORA_PACKAGE}

rm /var/lib/pacman/db.lck 2> /dev/null \
        || true

sudo -u nobody mkdir /app/build/$pkg

sudo -u nobody git clone https://aur.archlinux.org/$pkg.git /app/build/$pkg

cd /app/build/$pkg && sudo -u nobody -E makepkg --syncdeps --noconfirm

cp -r /app/build/$pkg/*.pkg.* /buffer

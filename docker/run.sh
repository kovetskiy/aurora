#!/bin/bash

set -euo pipefail

cd /app/build/$AURORA_PACKAGE

buildtime=$(date +%s)

sudo -u nobody -E makepkg --syncdeps --noconfirm

find ./ -maxdepth 1 -type f -name '*.pkg.*' -printf '%P\n' | while read filename; do
    cp "${filename}" "/buffer/$AURORA_PACKAGE/${buildtime}.${filename}"
done

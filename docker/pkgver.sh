#!/bin/bash

set -euo pipefail

. $(dirname "$0")/dir.sh

cp PKGBUILD PKGBUILD.pkgver

cat >> PKGBUILD.pkgver <<FUNC
	build() {
		echo -n "\$pkgver" > /app/build/$AURORA_PACKAGE/pkgver
		exit 0
	}
FUNC

chown nobody: PKGBUILD.pkgver
sudo -u nobody -E makepkg --syncdeps --noconfirm -p PKGBUILD.pkgver

cp /app/build/$AURORA_PACKAGE/pkgver /buffer/$AURORA_PACKAGE/pkgver
rm PKGBUILD.pkgver

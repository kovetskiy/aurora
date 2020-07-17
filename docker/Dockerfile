FROM archlinux/base

COPY /base /base
COPY /keys /keys

ARG CACHE=1

COPY /etc/pacman.conf /etc/pacman.conf
COPY /etc/pacman.d/mirrorlist /etc/pacman.d/mirrorlist

RUN gpg --recv-keys $(cat /keys)
RUN pacman -Sy --noconfirm $(cat /base)

RUN mkdir -p /app/build && \
    chgrp nobody /app/build && \
    chmod g+ws /app/build && \
    setfacl -m u::rwx,g::rwx /app/build && \
    setfacl -d --set u::rwx,g::rwx,o::- /app/build && \
    echo "nobody ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers

RUN chown -R nobody:nobody /root

COPY /etc/makepkg.conf /etc/makepkg.conf
COPY /etc/passwd /etc/passwd
COPY /etc/ssh/ssh_config /etc/ssh/ssh_config

COPY /etc/sudoers.d/nobody /etc/sudoers.d/nobody
RUN sed -ri 's/(nobody:.*):1:/\1::/' /etc/shadow

COPY /run.sh /app/run.sh
COPY /pkgver.sh /app/pkgver.sh
COPY /dir.sh /app/dir.sh

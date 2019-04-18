FROM archlinux/base

COPY /base /base
COPY /keys /keys

RUN DOCKER_CACHE=$(date +%s) yes | pacman -Sy $(cat /base)
RUN DOCKER_CACHE=$(date +%s) gpg --recv-keys $(cat /keys)

RUN mkdir -p /app/build && \
    chgrp nobody /app/build && \
    chmod g+ws /app/build && \
    setfacl -m u::rwx,g::rwx /app/build && \
    setfacl -d --set u::rwx,g::rwx,o::- /app/build && \
    echo "nobody ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers

RUN chown -R nobody:nobody /root

COPY /etc/pacman.conf /etc/pacman.conf
COPY /etc/pacman.d/mirrorlist /etc/pacman.d/mirrorlist
COPY /etc/passwd /etc/passwd
COPY /etc/ssh/ssh_config /etc/ssh/ssh_config

COPY /run.sh /app/run.sh

RUN DOCKER_CACHE=$(date +%s) pacman -Sy --noconfirm

CMD /app/run.sh

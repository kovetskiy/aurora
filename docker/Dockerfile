FROM archlinux/base

COPY /base /base
RUN yes | pacman -Sy $(cat /base)

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

RUN pacman -Sy --noconfirm

COPY /run.sh /app/run.sh

CMD /app/run.sh

FROM archlinux/base

RUN pacman -Sy --noconfirm git bzr grep awk openssh \
    make pkg-config diffutils bind-tools \
    dhcpcd iproute2 iputils \
    sudo gcc which gtk2 gtk3 fakeroot file

RUN mkdir -p /app/build && \
    chgrp nobody /app/build && \
    chmod g+ws /app/build && \
    setfacl -m u::rwx,g::rwx /app/build && \
    setfacl -d --set u::rwx,g::rwx,o::- /app/build && \
    echo "nobody ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers

COPY ./etc/pacman.conf /etc/pacman.conf
COPY ./etc/pacman.d/mirrorlist /etc/pacman.d/mirrorlist

COPY ./run.sh /app/run.sh

CMD /app/run.sh

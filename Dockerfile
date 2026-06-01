FROM golang:1.20

ENV GOPROXY https://goproxy.cn,direct

USER root
RUN touch /etc/apt/sources.list &&\
    sed -i 's/deb.debian.org/mirrors.tuna.tsinghua.edu.cn/' /etc/apt/sources.list && \
    sed -i 's/security.debian.org/mirrors.tuna.tsinghua.edu.cn/' /etc/apt/sources.list && \
    sed -i 's/security-cdn.debian.org/mirrors.tuna.tsinghua.edu.cn/' /etc/apt/sources.list && \
    apt update && \
    apt upgrade -y && \
    apt install -y --no-install-recommends &&\
    apt install -y libgmp-dev

COPY . /usr/src/code

USER root
RUN apt-get clean && \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/* && \
    rm /var/log/lastlog /var/log/faillog 

WORKDIR /usr/src/code

EXPOSE 8081
EXPOSE 8082
EXPOSE 8083
EXPOSE 8084

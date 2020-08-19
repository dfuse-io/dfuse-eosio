ARG EOSIO_TAG="v2.0.6-dm-12.0"
ARG DEB_PKG="eosio_2.0.6-dm.12.0-1-ubuntu-18.04_amd64.deb"

FROM ubuntu:18.04 AS base
ARG EOSIO_TAG
ARG DEB_PKG
RUN apt update && apt-get -y install curl ca-certificates libicu60 libusb-1.0-0 libcurl3-gnutls
RUN mkdir -p /var/cache/apt/archives/
RUN curl -sL -o/var/cache/apt/archives/eosio.deb "https://github.com/dfuse-io/eos/releases/download/${EOSIO_TAG}/${DEB_PKG}"
RUN dpkg -i /var/cache/apt/archives/eosio.deb
RUN rm -rf /var/cache/apt/*

FROM node:12 AS dlauncher
WORKDIR /work
RUN apt update && apt-get -y install git
RUN cd /work && git clone https://github.com/dfuse-io/dlauncher.git . &&\
    cd dashboard/client &&\
    yarn install && yarn build

FROM node:12 AS eosq
ADD eosq /work
WORKDIR /work
RUN yarn install && yarn build

FROM golang:1.14 as dfuse
RUN go get -u github.com/GeertJohan/go.rice/rice && export PATH=$PATH:$HOME/bin:/work/go/bin
RUN mkdir -p /work/build
ADD . /work
WORKDIR /work
COPY --from=eosq      /work/ /work/eosq
# The copy needs to be one level higher than work, the dashboard generates expects this file layout
COPY --from=dlauncher /work/ /dlauncher
RUN cd /dlauncher/dashboard && go generate
RUN cd /work/eosq/app/eosq && go generate
RUN cd /work/dashboard && go generate
RUN go test ./...
RUN go build -v -o /work/build/dfuseeos ./cmd/dfuseeos

FROM base
RUN mkdir -p /app/ && curl -Lo /app/grpc_health_probe https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/v0.2.2/grpc_health_probe-linux-amd64 && chmod +x /app/grpc_health_probe
COPY --from=dfuse /work/build/dfuseeos /app/dfuseeos
COPY --from=dfuse /work/tools/manageos/motd /etc/motd
COPY --from=dfuse /work/tools/manageos/scripts /usr/local/bin/
RUN echo cat /etc/motd >> /root/.bashrc
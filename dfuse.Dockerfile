ARG COMMIT=""
ARG VERSION=""

# NOTE: this for transittion when we still have multiple build with eosio
# should be removed in future
ARG UBUNTU_VERSION="22.04"
FROM ubuntu:${UBUNTU_VERSION} AS base
ARG UBUNTU_VERSION
ADD ./.github/install_deps.sh /
RUN ./install_deps.sh ${UBUNTU_VERSION}

FROM node:14 AS dlauncher
WORKDIR /work
ADD go.mod /work
RUN apt update && apt-get -y install git
RUN cd /work && git clone https://github.com/streamingfast/dlauncher.git dlauncher &&\
    grep -w github.com/streamingfast/dlauncher go.mod | sed 's/.*-\([a-f0-9]*$\)/\1/' |head -n 1 > dlauncher.hash &&\
    cd dlauncher &&\
    git checkout "$(cat ../dlauncher.hash)" &&\
    cd dashboard/client &&\
    yarn install --frozen-lockfile && yarn build

FROM node:14 AS eosq
ADD eosq /work
WORKDIR /work
RUN yarn install --frozen-lockfile && yarn build

FROM golang:1.20 as dfuse
ARG COMMIT
ARG VERSION
RUN mkdir -p /work/build
RUN mkdir -p /work/go/bin
ADD . /work
WORKDIR /work
RUN cp /work/go.rice/rice /work/go/bin/rice
ENV PATH="${PATH}:$HOME/bin:/work/go/bin"
RUN go install github.com/GeertJohan/go.rice/rice@latest
COPY --from=eosq      /work/ /work/eosq
# The copy needs to be one level higher than work, the dashboard generates expects this file layout
COPY --from=dlauncher /work/dlauncher /dlauncher
RUN cd /dlauncher/dashboard && go generate
RUN cd /work/eosq/app/eosq && go generate
RUN cd /work/dgraphql && go generate
RUN go test ./...
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" -v -o /work/build/dfuseeos ./cmd/dfuseeos

FROM base
RUN mkdir -p /app/ && curl -Lo /app/grpc_health_probe https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/v0.2.2/grpc_health_probe-linux-amd64 && chmod +x /app/grpc_health_probe
COPY --from=dfuse /work/build/dfuseeos /app/dfuseeos
COPY --from=dfuse /work/tools/manageos/motd /etc/motd
COPY --from=dfuse /work/tools/manageos/scripts /usr/local/bin/
RUN curl https://cmake.org/files/v3.13/cmake-3.13.4-Linux-x86_64.tar.gz | tar --strip-components=1 -xz -C /usr/local

RUN apt-get update && apt install -y librdkafka-dev
RUN echo cat /etc/motd >> /root/.bashrc

ENV PATH=$PATH:/app

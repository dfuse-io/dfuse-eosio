#!/usr/bin/env bash
set -e

apt update
if [ "$1" == "18.04" ] ; then
    apt-get -y install curl ca-certificates libicu60 libusb-1.0-0 libcurl3-gnutls
fi

if [ "$1" == "22.04" ] ; then
    apt-get -y install curl ca-certificates libc6 libgcc1 libstdc++6 libtinfo5 zlib1g libusb-1.0-0 libcurl3-gnutls
fi
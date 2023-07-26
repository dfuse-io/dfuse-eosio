ARG DFUSE_IMAGE=""
ARG DEB_PKG=""

FROM ${DFUSE_IMAGE}
ARG DEB_PKG
RUN mkdir -p /var/cache/apt/archives/
ADD ${DEB_PKG} /var/cache/apt/archives/
RUN dpkg -i /var/cache/apt/archives/${DEB_PKG}
RUN rm -rf /var/cache/apt/*

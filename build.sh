#!/bin/bash
CWD=`dirname "$0"`
FPATH=`( cd "$CWD" && pwd )`
VERSION=`git describe --tags`
NAME="dfuseeos-${VERSION}"
BINPATH=$FPATH/build/$NAME
printf "\nBuilding $NAME ...\n\n"
if go build -o $BINPATH  $FPATH/cmd/dfuseeos; then 
    printf '\e[1;32m%-6s\e[m\n' "Build Successful"
    printf "Created $BINPATH\n"
    printf "\nTo make 'dfuseeos' available system-wide:\n"
    printf "sudo cp $BINPATH /usr/local/bin/dfuseeos\n\n"
else 
    printf '\n\e[1;31m%-6s\e[m\n' "Build Failed"
fi

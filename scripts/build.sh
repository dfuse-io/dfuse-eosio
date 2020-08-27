#!/usr/bin/env bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && cd .. && pwd )"

force_build=
prepare_only=
skip_checks=
yes=

main() {
  pushd "$ROOT" &> /dev/null

  while getopts "hsyfp" opt; do
    case $opt in
      h) usage && exit 0;;
      f) force_build=true;;
      s) skip_checks=true;;
      p) prepare_only=true;;
      y) yes=true;;
      \?) usage_error "Invalid option: -$OPTARG";;
    esac
  done
  shift $((OPTIND-1))

  if [[ $skip_checks != true ]]; then
    if ! checks; then
      exit 1
    fi
  fi

  if ! build; then
    exit 1
  fi
}

build() {
  if [[ ! -d eosq/eosq-build || $force_build == true ]]; then
    pushd eosq > /dev/null
      echo "Building eosq"
      yarn install && yarn build
    popd > /dev/null
  fi

  dlauncher_hash=`grep -w github.com/dfuse-io/dlauncher go.mod | sed 's/.*-\([a-f0-9]*$\)/\1/' | head -n 1`
  pushd .. > /dev/null
    if [[ ! -d dlauncher ]]; then
      echo "Cloning dlauncher dependency"
      git clone https://github.com/dfuse-io/dlauncher
      git checkout $dlauncher_hash
    elif [[ $force_build == true ]]; then
      pushd dlauncher > /dev/null
        git pull
        git checkout $dlauncher_hash
      popd > /dev/null
    fi
    pushd dlauncher
      git checkout $DLAUNCHER
    popd >/dev/null

    if [[ ! -d dlauncher/dashboard/dashboard-build || $force_build == true ]]; then
      pushd dlauncher/dashboard > /dev/null
        pushd client > /dev/null
          echo "Buildind dashboard"
          yarn install && yarn build
        popd > /dev/null

        go generate .
      popd > /dev/null
    fi
  popd > /dev/null

  if [[ $force_build == true ]]; then
    echo "Generating static assets"
    go generate ./...
  fi

  if ! [[ $prepare_only == true ]]; then
    GIT_COMMIT="$(git describe --match=NeVeRmAtCh --always --abbrev=7 --dirty)"

    echo "Building & installing dfuseeos binary for $GIT_COMMIT"
    go install -ldflags "-X main.commit=$GIT_COMMIT" ./cmd/dfuseeos
  fi
}

checks() {
  found_error=
  if ! command -v go &> /dev/null; then
    echo "The 'go' command (version 1.14+) is required to build a version locally, install it following https://golang.org/doc/install#install"
    found_error=true
  else
    if ! (go version | grep -qE 'go1\.(1[456789]|[2-9][0-9]+)'); then
      echo "Your 'go' version (`go version`) is too low, requires go 1.14+, if you think it's a mistake, use '-s' flag to skip checks"
      found_error=true
    fi
  fi

  if ! command -v yarn &> /dev/null; then
    echo "The 'yarn' command (version 1.12+) is required to build a version locally, install it following https://classic.yarnpkg.com/en/docs/install"
    found_error=true
  else
    if ! (yarn --version | grep -qE '1\.(1[3456789]|[2-9][0-9])'); then
      echo "Your 'yarn' version (`yarn --version`) is too low, requires Yarn 1.12+, if you think it's a mistake, use '-s' flag to skip checks"
      found_error=true
    fi
  fi

  if ! command -v rice &> /dev/null; then
    install_rice=$yes
    if [[ $yes != true ]]; then
      if [ ! -t 0 ]; then
        echo "Terminal does not seem to accept user input, use the '-y' option to install by default"
      else
        read -p "The 'rice' executable was not found, do you want to install 'rice' now? (Y/N): " confirm
        if [[ $confirm == [yY] || $confirm == [yY][eE][sS] ]]; then
          install_rice=true
        fi
      fi
    fi

    if [[ $install_rice == true ]]; then
      pushd /tmp > /dev/null
        set -e
        echo "Installing 'rice' executable"
        go get github.com/GeertJohan/go.rice
        go get github.com/GeertJohan/go.rice/rice
        set +e
      popd > /dev/null
    else
      echo "The 'rice' executable is required to build a version locally, install it following https://github.com/GeertJohan/go.rice#installation"
      found_error=true
    fi
  fi

  if [[ $found_error == true ]]; then
    return 1
  fi
}

usage_error() {
  message="$1"
  exit_code="$2"

  echo "ERROR: $message"
  echo ""
  usage
  exit ${exit_code:-1}
}

usage() {
  echo "usage: ./scripts/build.sh [-f] [-s] [-y] [-p]"
  echo ""
  echo "Checks build requirements and build a development local version of the"
  echo "'dfuseeos' binary."
  echo ""
  echo "Options"
  echo "   -f          Force re-build all dependencies (eosq, dashboard)"
  echo "   -s          Skip all checks usually performed by this script"
  echo "   -y          Answers yes to all question asked by this script"
  echo "   -p          Prepare only all required artifacts for build, but don't run the build actually"
}

main "$@"

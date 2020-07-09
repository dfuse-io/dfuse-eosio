#!/usr/bin/env bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

clean=
active_pid=

finish() {
    kill -s TERM $active_pid &> /dev/null || true
}

main() {
  current_dir="`pwd`"
  trap "cd \"$current_dir\"" EXIT
  pushd "$ROOT" &> /dev/null

  while getopts "hc" opt; do
    case $opt in
      h) usage && exit 0;;
      c) clean=true;;
      \?) usage_error "Invalid option: -$OPTARG";;
    esac
  done
  shift $((OPTIND-1))

  compile_dfuseeos

  if [[ $clean == "true" ]]; then
    rm -rf dfuse-data &> /dev/null || true
  fi

  if [[ ! -d "dfuse-data" ]]; then
    dfuseeos -c injector.yaml start &
    active_pid=$!

    # We need to sleep more than really needed due to a "missing feature" in
    # fluxdb. Fluxdb does not flush its accumulated write on exit of the application
    # so writes are not flushed when not enough block has passed.
    sleep 10
    kill -s TERM $active_pid &> /dev/null
  fi

  dfuseeos -c server.yaml start
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
  echo "usage: start.sh [-c]"
  echo ""
  echo "Start $(basename $ROOT) environment."
  echo ""
  echo "Options"
  echo "    -c             Clean actual data directory first"
}

compile_dfuseeos() {
  pushd "$ROOT/../.." &> /dev/null
    go install ./cmd/dfuseeos
    if [[ $? != 0 ]]; then
      exit 1
    fi
  popd &> /dev/null
}

main "$@"

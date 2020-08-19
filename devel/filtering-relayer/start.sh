#!/usr/bin/env bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

dfuseeos="$ROOT/../dfuseeos"
clean=
all=true
only_filtering=
only_global=

finish() {
  for job in `jobs -p`; do
    kill -s TERM $job &> /dev/null || true
  done
}

main() {
  trap "finish" EXIT
  pushd "$ROOT" &> /dev/null

  while getopts "hcfg" opt; do
    case $opt in
      h) usage && exit 0;;
      c) clean=true;;
      f) all=false; only_filtering=true;;
      g) all=false; only_global=true;;
      \?) usage_error "Invalid option: -$OPTARG";;
    esac
  done
  shift $((OPTIND-1))

  if [[ $clean == "true" ]]; then
    rm -rf dfuse-data &> /dev/null || true
  fi

  echo "About to launch 2 apps, press Ctrl+C to terminal all jobs"
  echo "(This message is going to disappear in 2s)"
  sleep 2

  if [[ $all == true || $only_global == true ]]; then
    $dfuseeos -c global.yaml start &
  fi

  if [[ $all == true || $only_filtering == true ]]; then
    $dfuseeos -c filtering.yaml start &
  fi

  for job in `jobs -p`; do
    wait $job || true
  done
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
  echo "Environment"
  echo "    INFO=<app>     Turn info logs for <app> (multiple separated by ','), accepts app name or regexp (.* for all)"
  echo "    DEBUG=<app>    Turn debug logs for <app> (multiple separated by ','), accepts app name or regexp (.* for all)"
  echo ""
}

main "$@"
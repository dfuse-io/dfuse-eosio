#!/usr/bin/env bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

dfuseeos="$ROOT/../dfuseeos"
clean=
force_injection=
active_pid=

finish() {
    kill -s TERM $active_pid &> /dev/null || true
}

main() {
  trap "finish" EXIT
  pushd "$ROOT" &> /dev/null

  while getopts "hci" opt; do
    case $opt in
      h) usage && exit 0;;
      c) clean=true;;
      i) force_injection=true;;
      \?) usage_error "Invalid option: -$OPTARG";;
    esac
  done
  shift $((OPTIND-1))
  [[ $1 = "--" ]] && shift

  if [[ $clean == "true" ]]; then
    rm -rf dfuse-data 1> /dev/null
  fi

  set -e

  exec $dfuseeos -c server.yaml start "$@"
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
  echo "usage: start.sh [-c] [-i] [-- ... dfuseeos extra args]"
  echo ""
  echo "Start $(basename $ROOT) environment."
  echo ""
  echo "Options"
  echo "    -c             Clean actual data directory first"
  echo "    -i             Force injection, not just when no 'dfuse-data' present"
  echo ""
  echo "Environment"
  echo "    INFO=<app>     Turn info logs for <app> (multiple separated by ','), accepts app name or regexp (.* for all)"
  echo "    DEBUG=<app>    Turn debug logs for <app> (multiple separated by ','), accepts app name or regexp (.* for all)"
  echo ""
  echo "Examples"
  echo " Stream Actions        grpcurl -plaintext -d '{\"account\": 4157834734461014544}' localhost:9000 dfuse.eosio.accounthist.v1.AccountHistory.GetActions"
}

main "$@"

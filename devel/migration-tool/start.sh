#!/usr/bin/env bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

dfuseeos="$ROOT/../dfuseeos"
mode=
active_pid=

finish() {
    kill -s TERM $active_pid &> /dev/null || true
}

main() {
  trap "finish" EXIT
  pushd "$ROOT" &> /dev/null

  while getopts "hm:" opt; do
    case $opt in
      h) usage && exit 0;;
      m) mode="$OPTARG";;
      \?) usage_error "Invalid option: -$OPTARG";;
    esac
  done
  shift $((OPTIND-1))

  if [[ $mode == "export" ]]; then
    rm -rf migration-data
    $dfuseeos migrate -s "battlefield-snapshot.bin" "$@"
  elif [[ $mode == "import" ]]; then
    rm -rf dfuse-data
    $dfuseeos -c booter.yaml start "$@"
  else
    usage_error "You must specify either '-m export' or '-m import'"
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
  echo "usage: start.sh -m <mode>"
  echo ""
  echo "Start $(basename $ROOT) environment."
  echo ""
  echo "Options"
  echo "   -m export         Performs the export phase of the migration tool"
  echo "   -m import         Performs the import phase of the migration tool"
  echo ""
  echo "Environment"
  echo "    INFO=<app>     Turn info logs for <app> (multiple separated by ','), accepts app name or regexp (.* for all)"
  echo "    DEBUG=<app>    Turn debug logs for <app> (multiple separated by ','), accepts app name or regexp (.* for all)"
  echo ""
}

main $@
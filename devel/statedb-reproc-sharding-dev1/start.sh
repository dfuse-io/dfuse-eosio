#!/usr/bin/env bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

dfuseeos="$ROOT/../dfuseeos"
clean=
active_pid=

finish() {
    kill -s TERM $active_pid &> /dev/null || true
}

main() {
  trap "finish" EXIT
  pushd "$ROOT" &> /dev/null

  while getopts "hc" opt; do
    case $opt in
      h) usage && exit 0;;
      c) clean=true;;
      \?) usage_error "Invalid option: -$OPTARG";;
    esac
  done
  shift $((OPTIND-1))
  [[ $1 = "--" ]] && shift

  if [[ $clean == "true" ]]; then
    rm -rf dfuse-data 1> /dev/null
  fi

  set -e

  if [[ ! -d "dfuse-data" ]]; then
    # Each sharder generate all shards for a given range, this can be parallelize heavily as it depends only on `merged-blocks`
    echo "Generating statedb shards"
    $dfuseeos -c sharder-0-1000.yaml start "$@"
    echo ""
    $dfuseeos -c sharder-1000-2000.yaml start "$@"
    echo ""

    echo "Sharder is done"
    $dfuseeos tools check statedb-reproc-sharder dfuse-data/storage/statedb-shards 3
    echo ""

    # Injecting can be parallelize up to N where N is the number of generated shards (3 in this example)
    # Each injection instance runs sequentially for a given shard, but all shards can be injected in parallel.
    #
    # This theorical only, the underlying storage might like having 64 instances writing to it, so scale this
    # based on the throughput of your underlying storage engine. We usually runs like 8 to 16 in parallel on
    # on heavy to medium networks.
    echo "Injecting statedb shards into storage"
    $dfuseeos -c shard-injector-000.yaml start "$@"
    echo ""

    $dfuseeos -c shard-injector-001.yaml start "$@"
    echo ""

    $dfuseeos -c shard-injector-002.yaml start "$@"
    echo ""

    echo "Shard injector is done"
    echo ""
  fi

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
  echo "usage: start.sh [-c] [-- ... dfuseeos extra args]"
  echo ""
  echo "Start $(basename $ROOT) environment."
  echo ""
  echo "Options"
  echo "    -c             Clean actual data directory first"
  echo ""
  echo "Environment"
  echo "    INFO=<app>     Turn info logs for <app> (multiple separated by ','), accepts app name or regexp (.* for all)"
  echo "    DEBUG=<app>    Turn debug logs for <app> (multiple separated by ','), accepts app name or regexp (.* for all)"
  echo ""
}

main "$@"

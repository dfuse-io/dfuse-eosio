#!/usr/bin/env bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

dfuseeos="$ROOT/../dfuseeos"
clean=
read_only=

main() {
  pushd "$ROOT" &> /dev/null

  while getopts "hcr" opt; do
    case $opt in
      h) usage && exit 0;;
      c) clean=true;;
      r) read_only=true;;
      \?) usage_error "Invalid option: -$OPTARG";;
    esac
  done
  shift $((OPTIND-1))
  [[ $1 = "--" ]] && shift

  if [[ $clean == "true" ]]; then
    rm -rf dfuse-data &> /dev/null || true
  fi

  config_file=$(basename $ROOT).yaml
  if [[ $read_only ]]; then
    config_file="$(basename $ROOT)-readonly.yaml"
  fi

  exec $dfuseeos -c $config_file start "$@"
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
  echo "usage: start.sh [-c] [-r]"
  echo ""
  echo "Start $(basename $ROOT) environment."
  echo ""
  echo "Invocations"
  echo "  Starts a local StateDB connected to a bootstrapped local chain"
  echo "    start.sh"
  echo ""
  echo "  Starts a read-only StateDB connected to remote storage, useful to test read code against production data"
  echo '    start.sh -r -- --statedb-store-dsn="bigkv://<project>.<instance>/<table>"'
  echo ""
  echo "Options"
  echo "    -c             Clean actual data directory first"
  echo "    -r             Starts without a pipeline in read-only mode, easy to test read code on any network."
  echo ""
  echo "Environment"
  echo "    INFO=<app>     Turn info logs for <app> (multiple separated by ','), accepts app name or regexp (.* for all)"
  echo "    DEBUG=<app>    Turn debug logs for <app> (multiple separated by ','), accepts app name or regexp (.* for all)"
  echo ""
  echo ""
  echo "Examples (curl)"
  echo "    Health          curl http://localhost:8080/healthz"
  echo ""
  echo "Examples (grpc)"
  echo '    Stream Table    grpcurl -plaintext -d '\''{"contract":"eosio","table":"global","scope":"eosio","to_json":true}'\'' localhost:9000 dfuse.eosio.statedb.v1.State/StreamTableRows'
  echo '    Get Table Row   grpcurl -plaintext -d '\''{"contract":"eosio","table":"global","scope":"eosio","primary_key":"global","to_json":true}'\'' localhost:9000 dfuse.eosio.statedb.v1.State/GetTableRow'

}

main "$@"
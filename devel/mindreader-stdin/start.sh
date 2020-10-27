#!/usr/bin/env bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

dfuseeos="$ROOT/../dfuseeos"
clean=
all=true
only_mindreader=
only_producer=
only_stack=
use_deep_mind_file=

finish() {
  for job in `jobs -p`; do
    kill -s TERM $job &> /dev/null || true
  done

  wait
}

main() {
  trap "finish" EXIT
  pushd "$ROOT" &> /dev/null

  while getopts "hcmpsd:" opt; do
    case $opt in
      h) usage && exit 0;;
      c) clean=true;;
      m) all=false; only_mindreader=true;;
      p) all=false; only_producer=true;;
      s) all=false; only_stack=true;;
      d) use_deep_mind_file="$OPTARG";;
      \?) usage_error "Invalid option: -$OPTARG";;
    esac
  done
  shift $((OPTIND-1))
  [[ $1 = "--" ]] && shift

  if [[ $use_deep_mind_file != "" && ! -f $use_deep_mind_file ]]; then
    usage_error "The provided deep mind '$use_deep_mind_file' does not exist"
  fi

  if [[ $clean == "true" ]]; then
    rm -rf dfuse-data &> /dev/null || true
  fi

  if [[ $all == true ]]; then
    echo "About to launch 3 apps, press Ctrl+C to terminal all jobs"
    echo "(This message is going to disappear in 2s)"
    sleep 2
  fi

  if [[ $all == true || $only_producer == true ]]; then
    $dfuseeos -c producer.yaml start "$@" &
  fi

  if [[ $all == true || $only_mindreader == true ]]; then
    if [[ $use_deep_mind_file != "" ]]; then
      (cat $use_deep_mind_file | $dfuseeos -c mindreader-stdin.yaml start "$@") &
    else
      # Sleeping a few seconds to let the procuding node enough time to start
      sleep 3

      nodeos="nodeos --config-dir ./mindreader -d ./dfuse-data/mindreader/data --genesis-json=./mindreader/genesis.json --deep-mind"
      ($nodeos | $dfuseeos -c mindreader-stdin.yaml start "$@") &
    fi
  fi

  if [[ $all == true || $only_stack == true ]]; then
    $dfuseeos -c stack.yaml start "$@" &
  fi

  wait
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
  echo "usage: start.sh [-c] [-p] [-m] [-s] [-d <file>]"
  echo ""
  echo "Start $(basename $ROOT) environment."
  echo ""
  echo "Options"
  echo "    -c             Clean actual data directory first"
  echo "    -p             Only launch the producer app"
  echo "    -m             Only launch the mindreader app"
  echo "    -s             Only launch the stack apps (so everything expect producer & mindreader("
  echo "    -d <file>      Uses this deep ming log file (in '.dmlog' format) as the stdin pipe to the process instead of launching 'nodoes' process"
  echo ""
  echo "Environment"
  echo "    INFO=<app>     Turn info logs for <app> (multiple separated by ','), accepts app name or regexp (.* for all)"
  echo "    DEBUG=<app>    Turn debug logs for <app> (multiple separated by ','), accepts app name or regexp (.* for all)"
  echo ""
}

main "$@"
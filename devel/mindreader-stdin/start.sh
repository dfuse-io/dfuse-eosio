#!/usr/bin/env bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

clean=
all=true
only_mindreader=
only_producer=
use_deep_mind_file=

finish() {
  for job in `jobs -p`; do
    kill -s TERM $job &> /dev/null || true
  done

  wait
}

main() {
  current_dir="`pwd`"
  trap "cd \"$current_dir\"" EXIT
  pushd "$ROOT" &> /dev/null

  while getopts "hcmpd:" opt; do
    case $opt in
      h) usage && exit 0;;
      c) clean=true;;
      m) all=false; only_mindreader=true;;
      p) all=false; only_producer=true;;
      d) use_deep_mind_file="$OPTARG";;
      \?) usage_error "Invalid option: -$OPTARG";;
    esac
  done
  shift $((OPTIND-1))

  if [[ $use_deep_mind_file != "" && ! -f $use_deep_mind_file ]]; then
    usage_error "The provided deep mind '$use_deep_mind_file' does not exist"
  fi

  compile_dfuseeos

  if [[ $clean == "true" ]]; then
    rm -rf dfuse-data &> /dev/null || true
  fi

  if [[ $all == true ]]; then
    echo "About to launch 2 apps, press Ctrl+C to terminal all jobs"
    echo "(This message is going to disappear in 2s)"
    sleep 2
  fi

  if [[ $all == true || $only_producer == true ]]; then
    dfuseeos -c producer.yaml start &
  fi

  if [[ $all == true || $only_mindreader == true ]]; then
    if [[ $use_deep_mind_file != "" ]]; then
      (cat $use_deep_mind_file | dfuseeos -c mindreader-stdin.yaml start) &
    else
      # Sleeping a few seconds to let the procuding node enough time to start
      sleep 3

      nodeos="nodeos --config-dir ./mindreader -d ./dfuse-data/mindreader/data --genesis-json=./mindreader/genesis.json --deep-mind"
      ($nodeos | dfuseeos -c mindreader-stdin.yaml start) &
    fi
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
  echo "usage: start.sh [-c]"
  echo ""
  echo "Start $(basename $ROOT) environment."
  echo ""
  echo "Options"
  echo "    -c             Clean actual data directory first"
  echo "    -p             Only launch the producer app"
  echo "    -m             Only launch the mindreader app (and all others)"
  echo "    -d <file>      Uses this deep ming log file (in '.dmlog' format) as the stdin pipe to the process instead of launching 'nodoes' process"
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
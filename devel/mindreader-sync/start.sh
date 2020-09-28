#!/usr/bin/env bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

dfuseeos=../dfuseeos

clean=
network=
snapshot=
stop_block=

main() {
  current_dir="`pwd`"
  trap "cd \"$current_dir\"" EXIT
  pushd "$ROOT" &> /dev/null

  while getopts "hcn:s:e:" opt; do
    case $opt in
      h) usage && exit 0;;
      c) clean=true;;
      n) network="$OPTARG";;
      s) snapshot="$OPTARG";;
      e) stop_block="$OPTARG";;
      \?) usage_error "Invalid option: -$OPTARG";;
    esac
  done
  shift $((OPTIND-1))
  [[ $1 = "--" ]] && shift

  if [[ $network == "" || ! -d mindreader/$network ]]; then
    usage_error "Unknown network '$network', valid networks: `valid_networks`"
  fi

  if [[ $snapshot != "" && ! -f $snapshot ]]; then
    usage_error "Unknown snapshot file '$snapshot', are you sure it exists?"
  fi

  if [[ $clean == "true" ]]; then
    rm -rf dfuse-data &> /dev/null || true
  fi

  DFUSEEOS_MINDREADER_STOP_BLOCK_NUM=$stop_block\
  DFUSEEOS_MINDREADER_CONFIG_DIR=mindreader/$network \
  DFUSEEOS_MINDREADER_RESTORE_SNAPSHOT_NAME=$snapshot \
  $dfuseeos -c sync.yaml start "$@"
}

valid_networks() {
  to_joined_list `ls mindreader | sort`
}

to_joined_list() {
  echo "$@" | sed 's/ /, /g' | sed 's/, $//'
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
  echo "usage: start.sh [-c] -n <network> -s <snapshot> [-e <stopBlockNum>]"
  echo ""
  echo "Start $(basename $ROOT) syncing environment. This command requires you to provide '-n <network>' option"
  echo "to specify which network to sync with. If the network you want to sync with does not exist, simply create"
  echo "a new folder with the network name in './mindreader' folder (containing 'config.ini' and 'genesis.json' files)."
  echo ""
  echo "You can start from a given snapshot file simply by providing '-s <snapshot>' option. The snapshot must be"
  echo "already uncompress here. Moreover, you can make the instance stop at a given block by using "
  echo "'-e <stopBlock>' argument."
  echo ""
  echo "Required Options"
  echo "    -n                The network you want to sync with, valid values are: `valid_networks`"
  echo ""
  echo "Options"
  echo "    -c                Clean actual data directory first"
  echo "    -s <snapshot>     Define the snapshot file to use to start from"
  echo "    -e <stopBlock>    Define the stop block where to stop processing"
  echo ""
  echo "Environment"
  echo "    INFO=<app>     Turn info logs for <app> (multiple separated by ','), accepts app name or regexp (.* for all)"
  echo "    DEBUG=<app>    Turn debug logs for <app> (multiple separated by ','), accepts app name or regexp (.* for all)"
  echo ""
}

main "$@"
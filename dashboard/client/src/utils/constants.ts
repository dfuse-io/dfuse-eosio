// max drift value we display on the gaph
export const MAX_GRAPH_DRIFT = 15 * 60 * 60;
export const INFINITE_DRIFT_THRESHOLD = 9000000000;

// retain half an hour of data
export const DRIFT_RETENTION = 60;

export const MINDREADER_APP_ID = 'mindreader';

export type MetricConfig = {
  headBlockNumber: boolean;
  headBlockDrift: boolean;
};
export const METRIC_CONFIG: Record<string, MetricConfig> = {
  abicodec: {
    headBlockNumber: true,
    headBlockDrift: false
  },
  'search-archive': {
    headBlockNumber: true,
    headBlockDrift: false
  },
  blockmeta: {
    headBlockNumber: true,
    headBlockDrift: true
  },
  dashboard: {
    headBlockNumber: false,
    headBlockDrift: false
  },
  dgraphql: {
    headBlockNumber: false,
    headBlockDrift: false
  },
  'search-indexer': {
    headBlockNumber: true,
    headBlockDrift: true
  },
  'trxdb-loader': {
    headBlockNumber: true,
    headBlockDrift: true
  },
  'search-live': {
    headBlockNumber: true,
    headBlockDrift: true
  },
  mindreader: {
    headBlockNumber: true,
    headBlockDrift: true
  },
  merger: {
    headBlockNumber: true,
    headBlockDrift: true
  },
  relayer: {
    headBlockNumber: true,
    headBlockDrift: true
  },
  'search-router': {
    headBlockNumber: false,
    headBlockDrift: false
  },
  fluxdb: {
    headBlockNumber: true,
    headBlockDrift: true
  }
};

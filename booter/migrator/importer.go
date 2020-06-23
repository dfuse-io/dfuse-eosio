package migrator

import (
	"context"

	pbfluxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/fluxdb/v1"
)

type Exporter struct {
	ctx         context.Context
	fluxdb      pbfluxdb.StateClient
	exportDir   string
	irrBlockNum uint64

	notFoundCodes []string
	notFoundABIs  []string
	invalidABIs   []string
}

func NewExporter(ctx context.Context, fluxdb pbfluxdb.StateClient, exportDir string, irrBlockNum uint64) *Exporter {
	return &Exporter{
		ctx:         context.Background(),
		fluxdb:      pbfluxdb.NewStateClient(conn),
		exportDir:   exportDir,
		irrBlockNum: uint64(irrBlockNum),
	}
}

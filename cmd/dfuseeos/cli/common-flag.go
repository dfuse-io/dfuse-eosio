package cli

import (
	"time"

	"github.com/dfuse-io/dlauncher/launcher"
	"github.com/spf13/cobra"
)

func init() {
	launcher.RegisterCommonFlags = func(cmd *cobra.Command) error {
		// Common stores configuration flags
		cmd.Flags().String("common-backup-store-url", PitreosURL, "[COMMON] Store URL (with prefix) where to read or write backups.")
		cmd.Flags().String("common-blocks-store-url", MergedBlocksStoreURL, "[COMMON] Store URL (with prefix) where to read/write. Used by: relayer, fluxdb, trxdb-loader, blockmeta, search-indexer, search-live, search-forkresolver, eosws")
		cmd.Flags().String("common-oneblock-store-url", OneBlockStoreURL, "[COMMON] Store URL (with prefix) to read/write one-block files. Used by: mindreader, merger")
		cmd.Flags().String("common-blockstream-addr", RelayerServingAddr, "gRPC endpoint to get real-time blocks. Used by: fluxdb, trxdb-loader, blockmeta, search-indexer, search-live, eosws (relayer uses its own --relayer-blockstream-addr)")

		// Network config
		cmd.Flags().String("common-network-id", NetworkID, "Short network identifier, for billing purposes (usually maps namespaces on deployments). Used by: dgraphql")
		cmd.Flags().String("common-chain-id", "", "Chain ID in hex. Used by: trxdb-loader (to reverse the signatures and extract public keys)") // TODO: eventually, pluck that from somewhere instead of asking for it here (!). You risk noticing its missing very late, and it'll require reprocessing if you want the pubkeys.

		// Authentication, metering and rate limiter plugins
		cmd.Flags().String("common-auth-plugin", "null://", "Auth plugin URI, see dfuse-io/dauth repository")
		cmd.Flags().String("common-metering-plugin", "null://", "Metering plugin URI, see dfuse-io/dmetering repository")
		cmd.Flags().String("common-ratelimiter-plugin", "null://", "Rate Limiter plugin URI, see dfuse-io/dauth repository")

		// Database connection strings
		cmd.Flags().String("common-trxdb-dsn", TrxdbDSN, "kvdb connection string to trxdb database. Used by: trxdb-loader, abicodec, eosws, dgraphql")

		// Service addresses
		cmd.Flags().String("common-search-addr", RouterServingAddr, "gRPC endpoint to reach the Search Router. Used by: abicodec, eosws, dgraphql")
		cmd.Flags().String("common-blockmeta-addr", BlockmetaServingAddr, "gRPC endpoint to reach the Blockmeta. Used by: search-indexer, search-router, search-live, eosws, dgraphql")

		// Search flags
		// Register common search flags once for all the services
		cmd.Flags().String("search-common-mesh-store-addr", "", "[COMMON] Address of the backing etcd cluster for mesh service discovery.")
		cmd.Flags().String("search-common-mesh-dsn", DmeshDSN, "[COMMON] Dmesh DSN, supports local & etcd")
		cmd.Flags().String("search-common-mesh-service-version", DmeshServiceVersion, "[COMMON] Dmesh service version (v1)")
		cmd.Flags().Duration("search-common-mesh-publish-interval", 0*time.Second, "[COMMON] How often does search archive poll dmesh")
		cmd.Flags().String("search-common-action-filter-on-expr", "", "[COMMON] CEL program to whitelist actions to index. See https://github.com/dfuse-io/dfuse-eosio/blob/develop/search/README.md")
		cmd.Flags().String("search-common-action-filter-out-expr", "account == 'eidosonecoin' || receiver == 'eidosonecoin' || (account == 'eosio.token' && (data.to == 'eidosonecoin' || data.from == 'eidosonecoin'))", "[COMMON] CEL program to blacklist actions to index. These 2 options are used by search indexer, live and forkresolver.")
		cmd.Flags().String("search-common-dfuse-events-action-name", "", "[COMMON] The dfuse Events action name to intercept")
		cmd.Flags().Bool("search-common-dfuse-events-unrestricted", false, "[COMMON] Flag to disable all restrictions of dfuse Events specialize indexing, for example for a private deployment")
		cmd.Flags().String("search-common-indices-store-url", IndicesStoreURL, "[COMMON] Indices path to read or write index shards Used by: search-indexer, search-archiver.")

		return nil
	}
}

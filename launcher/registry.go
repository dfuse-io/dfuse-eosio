package launcher

import (
	"github.com/dfuse-io/dfuse-eosio/metrics"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var AppRegistry = map[string]*AppDef{}

func RegisterApp(appDef *AppDef) {
	AppRegistry[appDef.ID] = appDef
}

func GetMetricAppMeta() map[string]*metrics.AppMeta {
	mapping := make(map[string]*metrics.AppMeta)
	for _, appDef := range AppRegistry {
		mapping[appDef.MetricsID] = &metrics.AppMeta{
			Title: appDef.Title,
			Id:    appDef.ID,
		}
	}
	return mapping
}

func RegisterFlags(cmd *cobra.Command) error {
	for _, appDef := range AppRegistry {
		userLog.Debug("trying to register flags", zap.String("app_id", appDef.ID))
		if appDef.RegisterFlags != nil {
			userLog.Debug("found non nil flags, registering", zap.String("app_id", appDef.ID))
			err := appDef.RegisterFlags(cmd)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

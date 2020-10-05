package superviser

import (
	"regexp"
	"strings"

	logplugin "github.com/dfuse-io/node-manager/log_plugin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logLevelRegex = regexp.MustCompile("^(<[0-9]>)?(info|warn|error)")

func newToZapLogPlugin(debugDeepMind bool, logger *zap.Logger) *logplugin.ToZapLogPlugin {
	return logplugin.NewToZapLogPlugin(debugDeepMind, logger, logplugin.ToZapLogPluginLogLevel(logLevelExtractor))
}

var discardRegex = regexp.MustCompile("wabt.hpp:.*misaligned reference")
var toInfoRegex = regexp.MustCompile("(" +
	strings.Join([]string{
		"net_plugin.cpp:.*Closing connection to:",
		"controller.cpp:.*(No existing chain state or fork database|Initializing new blockchain with genesis state)",
		"platform_timer_accurac:.*Checktime timer",
	}, "|") +
	")")

func logLevelExtractor(in string) zapcore.Level {
	if discardRegex.MatchString(in) {
		return logplugin.NoDisplay
	}

	if toInfoRegex.MatchString(in) {
		return zap.InfoLevel
	}

	groups := logLevelRegex.FindStringSubmatch(in)
	if len(groups) <= 2 {
		return zap.DebugLevel
	}

	switch groups[2] {
	case "info":
		return zap.InfoLevel
	case "warn":
		return zap.WarnLevel
	case "error":
		return zap.ErrorLevel
	default:
		return zap.DebugLevel
	}
}

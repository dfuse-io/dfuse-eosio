// Copyright 2019 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dlauncher/flags"
	launcher "github.com/dfuse-io/dlauncher/launcher"
	"github.com/spf13/cobra"
)

// Root of the `dfuseeos` command
var RootCmd = &cobra.Command{Use: "dfuseeos", Short: "dfuse for EOSIO"}
var version = "dev"
var commit = ""
var allFlags = make(map[string]bool) // used as global because of async access to cobra init functions

func init() {
	RootCmd.Version = version + "-" + commit

}

func Main() {
	cobra.OnInitialize(func() {
		allFlags = flags.AutoBind(RootCmd, "DFUSEEOS")
	})

	RootCmd.PersistentFlags().StringP("data-dir", "d", "./dfuse-data", "Path to data storage for all components of dfuse")
	RootCmd.PersistentFlags().StringP("config-file", "c", "./dfuse.yaml", "dfuse configuration file to use. No config file loaded if set to an empty string.")
	RootCmd.PersistentFlags().String("nodeos-path", "nodeos", "Path to the nodeos binary. Defaults to the nodeos found in your PATH")
	RootCmd.PersistentFlags().Bool("skip-checks", false, "Skip checks to ensure 'nodeos' binary is supported")
	RootCmd.PersistentFlags().String("log-format", "text", "Format for logging to stdout. Either 'text' or 'stackdriver'")
	RootCmd.PersistentFlags().Bool("log-to-file", true, "Also write logs to {data-dir}/dfuse.log.json ")
	RootCmd.PersistentFlags().CountP("verbose", "v", "Enables verbose output (-vvvv for max verbosity)")

	RootCmd.PersistentFlags().String("log-level-switcher-listen-addr", "localhost:1065", "If non-empty, the process will listen on this address for json-formatted requests to change different logger levels (see LOGGING.md for more info)")
	RootCmd.PersistentFlags().String("metrics-listen-addr", MetricsListenAddr, "If non-empty, the process will listen on this address to server Prometheus metrics")
	RootCmd.PersistentFlags().String("pprof-listen-addr", "localhost:6060", "If non-empty, the process will listen on this address for pprof analysis (see https://golang.org/pkg/net/http/pprof/)")

	derr.Check("registering application flags", launcher.RegisterFlags(StartCmd))

	var availableCmds []string
	for app := range launcher.AppRegistry {
		availableCmds = append(availableCmds, app)
	}
	sort.Strings(availableCmds)

	StartCmd.SetHelpTemplate(fmt.Sprintf(startCmdHelpTemplate, strings.Join(availableCmds, "\n  ")))
	StartCmd.Example = startCmdExample

	RootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		return setupCmd(cmd)
	}

	derr.Check("dfuse", RootCmd.Execute())
}

var startCmdExample = `dfuseeos start relayer merger --merger-grpc-serving-addr=localhost:12345 --relayer-merger-addr=localhost:12345`
var startCmdHelpTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}} [all|command1 [command2...]]{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
  {{.Example}}{{end}}

Available applications:
  %s{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

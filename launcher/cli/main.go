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
	"github.com/abourget/viperbind"
	"github.com/dfuse-io/derr"
	"github.com/spf13/cobra"
)

// Root of the `dfuseeos` command
var RootCmd = &cobra.Command{Use: "dfuseeos", Short: "dfuse for EOSIO"}
var version = "dev"
var commit = ""

func init() {
	RootCmd.Version = version + "-" + commit

}

func Main() {
	cobra.OnInitialize(func() {
		viperbind.AutoBind(RootCmd, "DFUSEEOS")
	})

	RootCmd.PersistentFlags().StringP("data-dir", "d", "./dfusebox-data", "Path to data storage for all components of dfuse")
	RootCmd.PersistentFlags().StringP("config-file", "c", "./dfusebox.yaml", "dfusebox configuration file to use")
	RootCmd.PersistentFlags().String("nodeos-path", "nodeos", "Path to the nodeos binary. Defaults to the nodeos found in your PATH")
	RootCmd.PersistentFlags().CountP("verbose", "v", "Enables verbose output (-vvvv for max verbosity)")

	RootCmd.AddCommand(startCmd)
	RootCmd.AddCommand(purgeCmd)
	RootCmd.AddCommand(initCmd)

	RootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		setup()
	}

	derr.Check("dfusebox", RootCmd.Execute())
}

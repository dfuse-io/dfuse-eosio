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
	_ "net/http/pprof"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/spf13/cobra"
)

func Test_extractCmd(t *testing.T) {
	testCmdE := func(cmd *cobra.Command, args []string) error {
		return nil
	}

	rootCmd := &cobra.Command{Use: "dfuseeos", Short: "dfuse for EOSIO"}
	startCmd := &cobra.Command{Use: "start", Short: "Starts `dfuse for EOSIO` services all at once", RunE: testCmdE}
	initCmd := &cobra.Command{Use: "init", Short: "Initializes dfuse's local environment", RunE: testCmdE}
	toolCmd := &cobra.Command{Use: "tools", Short: "Developer tools related to dfuseeos", RunE: testCmdE}
	dbBlkCmd := &cobra.Command{Use: "blk", Short: "Read a Blk", RunE: testCmdE}

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(toolCmd)
	toolCmd.AddCommand(dbBlkCmd)

	tests := []struct {
		name      string
		cmd       *cobra.Command
		expectCmd []string
	}{
		{
			name:      "root command",
			cmd:       rootCmd,
			expectCmd: []string{"dfuseeos"},
		},
		{
			name:      "first tier command",
			cmd:       startCmd,
			expectCmd: []string{"dfuseeos", "start"},
		},
		{
			name:      "child command",
			cmd:       dbBlkCmd,
			expectCmd: []string{"dfuseeos", "tools", "blk"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectCmd, extractCmd(test.cmd))
		})
	}

}

func Test_shouldRunSetup(t *testing.T) {
	testCmdE := func(cmd *cobra.Command, args []string) error {
		return nil
	}

	rootCmd := &cobra.Command{Use: "dfuseeos", Short: "dfuse for EOSIO"}
	startCmd := &cobra.Command{Use: "start", Short: "Starts `dfuse for EOSIO` services all at once", RunE: testCmdE}
	initCmd := &cobra.Command{Use: "init", Short: "Initializes dfuse's local environment", RunE: testCmdE}
	toolCmd := &cobra.Command{Use: "tools", Short: "Developer tools related to dfuseeos", RunE: testCmdE}
	dbBlkCmd := &cobra.Command{Use: "blk", Short: "Read a Blk", RunE: testCmdE}

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(toolCmd)
	toolCmd.AddCommand(dbBlkCmd)

	tests := []struct {
		name       string
		cmds       []string
		expectBool bool
	}{
		{
			name:       "root command",
			cmds:       []string{"dfuseeos"},
			expectBool: false,
		},
		{
			name:       "first tier command",
			cmds:       []string{"dfuseeos", "start"},
			expectBool: true,
		},
		{
			name:       "child command",
			cmds:       []string{"dfuseeos", "tools", "blk"},
			expectBool: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectBool, shouldRunSetup(test.cmds, []*cobra.Command{
				StartCmd,
			}))
		})
	}

}

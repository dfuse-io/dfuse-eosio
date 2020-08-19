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
	"os"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var purgeCmd = &cobra.Command{Use: "purge", Short: "Purges dfuse's local data", RunE: dfusePurgeE}

func init() {
	RootCmd.AddCommand(purgeCmd)
	purgeCmd.Flags().BoolP("force", "f", false, "Force purging of data without user intervention")
}

func dfusePurgeE(cmd *cobra.Command, args []string) error {
	dataDir := viper.GetString("global-data-dir")

	if err := Purge(dataDir); err != nil {
		return err
	}

	userLog.Printf("Purged data. Start a fresh instance with 'dfuseeos start'")

	return nil
}

func Purge(dataDir string) error {
	purge, err := confirmPurgeAll(dataDir)
	if err != nil {
		return fmt.Errorf("unable to purge environment %w", err)
	}

	if purge {
		if err := os.RemoveAll(dataDir); err != nil {
			return fmt.Errorf("unable to correcty delete directory %q: %w", dataDir, err)
		}
	}

	return nil
}

func confirmPurgeAll(dataDir string) (bool, error) {
	if viper.GetBool("force") {
		return true, nil
	}

	prompt := promptui.Prompt{
		Label:     fmt.Sprintf("You are about to delete %q. Are you sure", dataDir),
		IsConfirm: true,
	}

	result, err := prompt.Run()

	if err != nil {
		return false, err
	}
	return strings.ToLower(result) == "y", nil
}

package cli

import (
	"fmt"
	"os/exec"
)

func CheckNodeosInstallation(path string) error {
	if path, err := exec.LookPath(path); err != nil {
		return fmt.Errorf("WARN: could not find %s (not in PATH).  Check https://github.com/dfuse-io/dfuse-eosio/blob/develop/INSTALL.md for instructions.", path)
	}

	return nil
}

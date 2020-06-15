package boot

import (
	"fmt"
	"os"

	"github.com/eoscanada/eos-go"
	eoscvault "github.com/eoscanada/eosc/vault"
	"github.com/spf13/viper"
)

func (b *booter) newKeyBagFromVault(vaultFile string) (*eos.KeyBag, error) {
	if _, err := os.Stat(vaultFile); err != nil {
		return nil, fmt.Errorf("vault file %q missing: %s", vaultFile, err)
	}

	vault, err := eoscvault.NewVaultFromWalletFile(vaultFile)
	if err != nil {
		return nil, fmt.Errorf("loading vault: %s", err)
	}

	boxer, err := eoscvault.SecretBoxerForType(vault.SecretBoxWrap, viper.GetString("global-kms-gcp-keypath"))
	if err != nil {
		return nil, fmt.Errorf("secret boxer: %s", err)
	}

	if err := vault.Open(boxer); err != nil {
		return nil, err
	}

	return vault.KeyBag, nil
}

package ultraops

import (
	"fmt"

	"github.com/dfuse-io/eosio-boot/config"
	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/ecc"
)

var AN = eos.AN
var ActN = eos.ActN
var PN = eos.PN

func decodeOpPublicKey(c *config.OpConfig, opPubKey string) (ecc.PublicKey, error) {
	privateKey, err := c.GetPrivateKey(opPubKey)
	if err == nil {
		return privateKey.PublicKey(), nil
	}

	pubKey, err := ecc.NewPublicKey(opPubKey)
	if err != nil {
		return ecc.PublicKey{}, fmt.Errorf("reading pubkey: %s", err)
	}
	return pubKey, nil
}

// // this is use to support ephemeral key
// func getBootKey(c *config.OpConfig) (ecc.PublicKey, error) {
// 	privateKey, err := c.GetPrivateKey("boot")
// 	if err == nil {
// 		return privateKey.PublicKey(), nil
// 	}

// 	privateKey, err = c.GetPrivateKey("ephemeral")
// 	if err == nil {
// 		return privateKey.PublicKey(), nil
// 	}

// 	return ecc.PublicKey{}, fmt.Errorf("cannot find boot/ephemeral key")
// }

// func retrieveABIfromRef(abiPath string) (*eos.ABI, error) {
// 	abiContent, err := ioutil.ReadFile(abiPath)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if len(abiContent) == 0 {
// 		return nil, fmt.Errorf("unable to unmarshal abi with 0 bytes")
// 	}

// 	var abiDef eos.ABI
// 	if err := json.Unmarshal(abiContent, &abiDef); err != nil {
// 		return nil, fmt.Errorf("unmarshal ABI file: %s", err)
// 	}

// 	return &abiDef, nil
// }

package migrator

import (
	"fmt"
	"io/ioutil"
	"strconv"

	"github.com/eoscanada/eos-go"

	rice "github.com/GeertJohan/go.rice"
)

var AN = eos.AN
var PN = eos.PN
var ActN = eos.ActN

func TN(in string) eos.TableName { return eos.TableName(in) }
func SN(in string) eos.ScopeName { return eos.ScopeName(in) }

func UINT64(in string) uint64 {
	v, err := strconv.ParseUint(in, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("cannot convert %q to uint64", in))
	}
	return v

}

func readBoxFile(box *rice.Box, filename string) ([]byte, error) {
	f, err := box.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("unable to open migration %q: %w", filename, err)
	}
	defer f.Close()
	cnt, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("unable to read box file %q content: %w", filename, err)
	}
	return cnt, nil
}

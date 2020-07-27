package migrator

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	pbts "github.com/golang/protobuf/ptypes/timestamp"

	"github.com/eoscanada/eos-go"
	"github.com/golang/protobuf/ptypes"

	rice "github.com/GeertJohan/go.rice"
)

var AN = eos.AN
var PN = eos.PN
var ActN = eos.ActN

func TN(in string) eos.TableName { return eos.TableName(in) }
func SN(in string) eos.ScopeName { return eos.ScopeName(in) }

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

func writeJSONFile(filename string, v interface{}) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	return encoder.Encode(v)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}

	if err != nil {
		return false
	}

	return !info.IsDir()
}

func mustTimePointToProto(point eos.TimePoint) *pbts.Timestamp {
	t := time.Unix(0, int64(point*1000))
	time, err := ptypes.TimestampProto(t)
	if err != nil {
		panic(fmt.Sprintf("invalid timestamp conversion %q: %s", time, err))
	}
	return time
}

package tools

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap/zapcore"

	"github.com/paulbellamy/ratecounter"
	"github.com/spf13/viper"
)

func getBlockRangeFromFlag() (out BlockRange, err error) {
	stringRange := viper.GetString("range")
	if stringRange == "" {
		return
	}

	rawRanges := strings.Split(stringRange, ",")
	if len(rawRanges) == 0 {
		return
	}

	if len(rawRanges) > 1 {
		return out, fmt.Errorf("accepting a single range for now, got %d", len(rawRanges))
	}

	out, err = decodeRange(rawRanges[0])
	if err != nil {
		return out, fmt.Errorf("decode range: %w", err)
	}

	return
}

type BlockRange struct {
	Start uint64
	Stop  uint64
}

func (b BlockRange) Unbounded() bool {
	return b.Start == 0 && b.Stop == 0
}

func (b BlockRange) ReprocRange() string {
	return fmt.Sprintf("%d:%d", b.Start, b.Stop+1)
}

func (b BlockRange) String() string {
	return fmt.Sprintf("%s - %s", blockNum(b.Start), blockNum(b.Stop))
}

func decodeRanges(rawRanges string) (out []BlockRange, err error) {
	for _, rawRange := range strings.Split(rawRanges, ",") {
		blockRange, err := decodeRange(rawRange)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}

		out = append(out, blockRange)
	}

	return
}

func decodeRange(rawRange string) (out BlockRange, err error) {
	parts := strings.SplitN(rawRange, ":", 2)
	if len(parts) != 2 {
		return out, fmt.Errorf("invalid range %q, not matching format `<start>:<end>`", rawRange)
	}

	out.Start, err = decodeBlockNum("start", parts[0])
	if err != nil {
		return
	}

	out.Stop, err = decodeBlockNum("end", parts[1])
	if err != nil {
		return
	}

	return
}

func decodeBlockNum(tag string, part string) (out uint64, err error) {
	trimmedValue := strings.Trim(part, " ")

	if trimmedValue != "" {
		out, err = strconv.ParseUint(trimmedValue, 10, 64)
		if err != nil {
			return out, fmt.Errorf("`<%s>` value %q is not a valid integer", tag, part)
		}

		if out < 0 {
			return out, fmt.Errorf("`<%s>` value %q should be positive (or 0)", tag, part)
		}
	}

	return
}

type FilteringFilters struct {
	Include string
	Exclude string
	System  string
}

func (f *FilteringFilters) Key() string {
	return f.System + f.Exclude + f.System
}

type stateFile struct {
	StartBlock uint64
	Source     string
}

type stats struct {
	startTime        time.Time
	timeToFirstBlock time.Duration
	blockReceived    *counter
	bytesReceived    *counter
	restartCount     *counter
}

func newStats() *stats {
	return &stats{
		startTime:     time.Now(),
		blockReceived: &counter{0, ratecounter.NewRateCounter(1 * time.Second), "block", "s"},
		bytesReceived: &counter{0, ratecounter.NewRateCounter(1 * time.Second), "byte", "s"},
		restartCount:  &counter{0, ratecounter.NewRateCounter(1 * time.Minute), "restart", "m"},
	}
}

func (s *stats) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddString("block", s.blockReceived.String())
	encoder.AddString("bytes", s.bytesReceived.String())
	return nil
}

func (s *stats) duration() time.Duration {
	return time.Now().Sub(s.startTime)
}

func (s *stats) recordBlock(payloadSize int64) {
	if s.timeToFirstBlock == 0 {
		s.timeToFirstBlock = time.Now().Sub(s.startTime)
	}

	s.blockReceived.IncBy(1)
	s.bytesReceived.IncBy(payloadSize)
}

type counter struct {
	total    uint64
	counter  *ratecounter.RateCounter
	unit     string
	timeUnit string
}

func (c *counter) IncBy(value int64) {
	if value <= 0 {
		return
	}

	c.counter.Incr(value)
	c.total += uint64(value)
}

func (c *counter) Total() uint64 {
	return c.total
}

func (c *counter) Rate() int64 {
	return c.counter.Rate()
}

func (c *counter) String() string {
	return fmt.Sprintf("%d %s/%s (%d total)", c.counter.Rate(), c.unit, c.timeUnit, c.total)
}

func (c *counter) Overall(elapsed time.Duration) string {
	rate := float64(c.total)
	if elapsed.Minutes() > 1 {
		rate = rate / elapsed.Minutes()
	}

	return fmt.Sprintf("%d %s/%s (%d %s total)", uint64(rate), c.unit, "min", c.total, c.unit)
}

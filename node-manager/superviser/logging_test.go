package superviser

import (
	"testing"

	"github.com/streamingfast/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToZapLogPlugin_LogLevel(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  string
	}{
		// The standard `nodeos` output
		{
			"debug",
			"debug  2020-10-05T13:53:11.749 thread-0  http_plugin.cpp:895 ...",
			`{"level":"debug","msg":"debug  2020-10-05T13:53:11.749 thread-0  http_plugin.cpp:895 ..."}`,
		},
		{
			"info",
			"info  2020-10-05T13:53:11.749 thread-0  http_plugin.cpp:895 ...",
			`{"level":"info","msg":"info  2020-10-05T13:53:11.749 thread-0  http_plugin.cpp:895 ..."}`,
		},
		{
			"warn",
			"warn  2020-10-05T13:53:11.749 thread-0  http_plugin.cpp:895 ...",
			`{"level":"warn","msg":"warn  2020-10-05T13:53:11.749 thread-0  http_plugin.cpp:895 ..."}`,
		},
		{
			"error",
			"error  2020-10-05T13:53:11.749 thread-0  http_plugin.cpp:895 ...",
			`{"level":"error","msg":"error  2020-10-05T13:53:11.749 thread-0  http_plugin.cpp:895 ..."}`,
		},
		{
			"other",
			"other  2020-10-05T13:53:11.749 thread-0  http_plugin.cpp:895 ...",
			`{"level":"debug","msg":"other  2020-10-05T13:53:11.749 thread-0  http_plugin.cpp:895 ..."}`,
		},

		// The weird `nodeos` output where some markup is appended
		{
			"weird markup, debug",
			"<0>debug  2020-10-05T13:53:11.749 thread-0  http_plugin.cpp:895 ...",
			`{"level":"debug","msg":"<0>debug  2020-10-05T13:53:11.749 thread-0  http_plugin.cpp:895 ..."}`,
		},
		{
			"weird markup, info",
			"<6>info  2020-10-05T13:53:11.749 thread-0  http_plugin.cpp:895 ...",
			`{"level":"info","msg":"<6>info  2020-10-05T13:53:11.749 thread-0  http_plugin.cpp:895 ..."}`,
		},
		{
			"weird markup, warn",
			"<4>warn  2020-10-05T13:53:11.749 thread-0  http_plugin.cpp:895 ...",
			`{"level":"warn","msg":"<4>warn  2020-10-05T13:53:11.749 thread-0  http_plugin.cpp:895 ..."}`,
		},
		{
			"weird markup, error",
			"<3>error  2020-10-05T13:53:11.749 thread-0  http_plugin.cpp:895 ...",
			`{"level":"error","msg":"<3>error  2020-10-05T13:53:11.749 thread-0  http_plugin.cpp:895 ..."}`,
		},
		{
			"weird markup, other",
			"[random]other  2020-10-05T13:53:11.749 thread-0  http_plugin.cpp:895 ...",
			`{"level":"debug","msg":"[random]other  2020-10-05T13:53:11.749 thread-0  http_plugin.cpp:895 ..."}`,
		},

		{
			"discarded wabt reference",
			"warn  2020-10-05T13:53:11.749 thread-5  wabt.hpp:106	misaligned reference",
			``,
		},
		{
			"to info closing connection to",
			"error  2020-10-05T13:53:11.749 thread-0  net_plugin.cpp:253 Closing connection to: 192.168.0.189",
			`{"level":"info","msg":"error  2020-10-05T13:53:11.749 thread-0  net_plugin.cpp:253 Closing connection to: 192.168.0.189"}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			wrapper := logging.NewTestLogger(t)
			plugin := newToZapLogPlugin(false, wrapper.Instance())
			plugin.LogLine(test.in)

			loggedLines := wrapper.RecordedLines(t)

			if len(test.out) == 0 {
				require.Len(t, loggedLines, 0)
			} else {
				require.Len(t, loggedLines, 1)
				assert.Equal(t, test.out, loggedLines[0])
			}
		})
	}
}

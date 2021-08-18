package tools

import "github.com/streamingfast/cli"

var Example = func(in string) string {
	return string(cli.Example(in))
}

var ExamplePrefixed = func(prefix, in string) string {
	return string(cli.ExamplePrefixed(prefix, in))
}

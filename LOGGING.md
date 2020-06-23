# Logging

It's possible to configure `dfuseeos` logging with the repeatable
verbosity flag, like `-vvv` which enables debugging verbosity level
3 (default is 0).

Below you will find the default logging level per package(s) for various verbosity
levels as well as formatting rules in place depending on the verbosity level.

As the verbosity level increases, the more debugging statements are displayed 
and log line will contain more contextual information to help you debug.

### Logs produced by nodeos (mindreader and node-manager dfuse apps)

By default, the logs produced by nodeos (node-manager or mindreader) will be 
processed through the dfuse logging system, which does not show them with lower 
verbosity levels (it makes the console unreadable when all applications are run together).

* To prevent them from being transformed and gated by the dfuse logger, you can use the two following flags:
  `--node-manager-log-to-zap=false` and `--mindreader-log-to-zap=false`

* To show all "info" log messages from mindreader and node-manager, you can 
use the environment variable INFO (detailed later in this document): INFO=

## Global Verbosity flags

These are the simplest way to manage verbosity in a holistic way.
Different levels are already defined to make it easier to run all
`dfuseeos` apps together.

### Verbosity 0 (no flag)

Level:

- INFO `github.com/dfuse-io/dfuse-eosio`
- INFO `github.com/dfuse-io/dfuse-eosio/cmd/dfuseeos`
- WARN All others

Formatting:

- No level displayed
- No logger name displayed
- Caller displayed when log entry level >= WARN
- Stacktrace displayed when log entry level >= ERROR (if present)

### Verbosity 1 (-v)

Level:

- INFO `github.com/dfuse-io/dfuse-eosio`
- INFO `github.com/dfuse-io/dfuse-eosio/cmd/dfuseeos`
- WARN `github.com/dfuse-io/manageos.*`
- INFO All apps
- WARN All others

Formatting:

- Level always displayed
- Logger name always displayed
- Caller always displayed, but package version trimmed out
- Stacktrace always displayed (if present)

### Verbosity 2 (-vv)

Level:

- DEBUG `github.com/dfuse-io/dfuse-eosio`
- DEBUG `github.com/dfuse-io/dfuse-eosio/cmd/dfuseeos`
- INFO `github.com/dfuse-io/manageos.*`
- INFO All apps
- INFO All others

Formatting:

- Level always displayed
- Logger name always displayed
- Caller always displayed, but package version trimmed out
- Stacktrace always displayed (if present)

### Verbosity 3 (-vvv)

Level:

- DEBUG All packages

Formatting:

- Level always displayed
- Logger name always displayed
- Caller always displayed, but package version trimmed out
- Stacktrace always displayed (if present)

### Verbosity 4+ (-vvvv [and more])

Level:

- DEBUG All packages

Formatting:

- Level always displayed
- Logger name always displayed
- Caller always displayed, full path, with package version present
- Stacktrace always displayed (if present)

## Specify Verbosity of dfuse apps or loggers with Environment Variables

* The `DEBUG`, `INFO` and `WARN` environment variables can be used to set the verbosity for specific app(s) value.

* As soon as `DEBUG` or `INFO` is set, the formatting rules will be set to the more verbose one:
- Level always displayed
- Logger name always displayed
- Caller always displayed, but package version trimmed out (unless -vvvv or more present)
- Stacktrace always displayed (if present)

* The value of those variable is comma-separated list of either **dfuse app name** ex: `search` or `search,mindreader`) or **logging module regular expressions** (ex: `github.com/dfuse-io/manageos.*nodeos`)

For example, you can run:

```
DEBUG="mindreader,dgraphql,relayer" INFO="mindreader/nodeos" dfuseeos start
```

which will:
1. set verbosity of apps `mindreader`, `dgraphql` and relayer to`DEBUG` level.
2. set verbosity of nodeos module to INFO (the string matches the logger registered as `github.com/dfuse-io/manageos/app/nodeos_mindreader/nodeos` in the code)

Note that logger matching will be applied in this order: WARN, INFO, DEBUG

### Changing log levels at runtime

You can switch the log levels of a given component by sending an HTTP request on port 1065 (configurable via --log-level-switcher-listen-addr flag) like this:

```
curl localhost:1065 -XPOST -d '{"level": "debug","inputs":"bstream"}'
curl localhost:1065 -XPOST -d '{"level": "info","inputs":".*"}'
curl localhost:1065 -XPOST -d '{"level": "warn","inputs":"merger,bstream,manageos,mindreader"}'
```

* Inputs expects the same format as environment variables for those levels, as described above.
* New log levels are always applied on top of previous ones (ex: the 'info' level for inputs `.*` would override any previous logging setting)


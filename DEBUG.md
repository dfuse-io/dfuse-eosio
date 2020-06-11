# Logging

It's possible to configure `dfuseeos` logging with the repeatable
verbosity flag, like `-vvv` which enables debugging verbosity level
3 (default is 0).

Here the default level per package(s) for the various verbosity level
as well as formatting rules in place depending on the verbosity level.

More higher is the verbosity level, more debugging statements and more
each log line has contextual information to help debugging.

### Logs produced by nodeos (mindreader and node-manager dfuse apps)

By default, the logs produced by nodeos (node-manager or mindreader) will be processed through the dfuse logging system, which does not show them with lower verbosity levels (it makes the console unreadable when all applications are run together).

* To prevent them from being transformed and gated by the dfuse logger, you can use the two following flags:
  `--node-manager-log-to-zap=false` and `--mindreader-log-to-zap=false`

* To show all "info" log messages from mindreader and node-manager, you can use the environment variable INFO (detailed later in this document): INFO=

## Global Verbosity flags

These are the simplest way to manage verbosity in a holistic way.
Different levels are already defined to make it easier to run all
`dfuseeos` apps together.

### Verbosity 0 (no flag)

Level:

- INFO `github.com/dfuse-io/dfuse-eosio`
- INFO `github.com/dfuse-io/dfuse-eosio/cmd/dfuseeos`
- <Hidden> All others

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
- INFO All others

Formatting:

- Level always displayed
- Logger name always displayed
- Caller displayed when log entry level >= WARN
- Stacktrace displayed when log entry level >= ERROR (if present)

### Verbosity 2 (-vv)

Level:

- DEBUG `github.com/dfuse-io/dfuse-eosio`
- DEBUG `github.com/dfuse-io/dfuse-eosio/cmd/dfuseeos`
- INFO `github.com/dfuse-io/manageos.*`
- INFO All others

Formatting:

- Level always displayed
- Logger name always displayed
- Caller displayed when log entry level >= WARN
- Stacktrace always displayed (if present)

### Verbosity 3 (-vvv)

Level:

- DEBUG All packages

Formatting:

- Level always displayed
- Logger name always displayed
- Caller always displayed, but package version trimmed out
- Stacktrace always displayed (if present)

#### Verbosity 4+ (-vvvv [and more])

Level:

- DEBUG All packages

Formatting:

- Level always displayed
- Logger name always displayed
- Caller always displayed, full path, with package version present
- Stacktrace always displayed (if present)

### Environment variable `DEBUG="app1,app2"`

Overrides behavior of verbosity for specific app(s) value
as well as changing the formatting rules. For example, you can run:

```
DEBUG="mindreader,dgraphql" dfuseeos start
```

Which will keep the level behavior of verbosity 0 but will set loggers
of app `mindreader` and `dgraphql` to `DEBUG` level.

The value can also be a regular expression, in which case it matches the
logger registration performed by a single package.

Formatting:

- Level always displayed
- Logger name always displayed
- Caller always displayed, but package version trimmed out (unless -vvvv or more present)
- Stacktrace always displayed (if present)

#### Changing log levels at runtime

You can switch the log levels of a given component by sending an HTTP request on port 1065 (configurable via --log-level-switcher-listen-addr flag) like this:

```
curl localhost:1065 -XPOST -d '{"level": "debug","inputs":"bstream"}'
curl localhost:1065 -XPOST -d '{"level": "info","inputs":".*"}'
curl localhost:1065 -XPOST -d '{"level": "warn","inputs":"merger,bstream,manageos,mindreader"}'
```

Values for inputs are always applied on top of previous ones.


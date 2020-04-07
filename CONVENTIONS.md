# Coding conventions

In general, this project adheres to all the standard Go conventions,
and is constantly on the look to make sure we have a uniform and
coherent codebase.

Here are things that might be useful to clarify nonetheless:


## Error management

* All `fmt.Errorf()` ought to start with a lowercase letter, with `:`
  separating elements, from broadest to most refined.

## Logging

* Logging is done through the `zap` library.  All developer-centric logs ought to start with a lowercase, and provide sufficient context to aid in debugging.
* Assume systems log at `Info` level by default, enabling `Debug` when needed (at runtime through port :1065 <- logs, you read that l33t?).
* Assume systems trigger some sort of alerting when `Warn` and `Error` level errors are triggered.

# The `dfuseeos` dashboard

This is the UI available when you start `dfuseeos`

## Development

This package uses https://github.com/GeertJohan/go.rice to embed the
static files within the Go binary.

Install with:

    go get github.com/GeertJohan/go.rice/rice

Generate the new `rice-box.go`:

    go generate

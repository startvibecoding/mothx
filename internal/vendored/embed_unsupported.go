//go:build !((linux && (amd64 || arm64)) || (darwin && (amd64 || arm64)) || (windows && (amd64 || arm64)))

package vendored

var rgData []byte
var fdData []byte

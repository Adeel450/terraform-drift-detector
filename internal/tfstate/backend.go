package tfstate

import (
	"fmt"
	"strings"
)

// BackendFactory builds a StateSource from a reference and optional region.
type BackendFactory func(ref, region string) (StateSource, error)

// backends maps a URI scheme (e.g. "s3") to its factory. The local-file backend
// is the default and handles refs without a recognized scheme.
var backends = map[string]BackendFactory{}

// RegisterBackend associates a scheme with a factory. Called from backend
// implementation init() functions (e.g. the S3 backend registers "s3").
func RegisterBackend(scheme string, f BackendFactory) {
	backends[scheme] = f
}

// Open builds a StateSource for ref. A ref of the form "<scheme>://..." is
// dispatched to the registered backend for that scheme; anything else is a
// local file path. region is passed through to backends that need it.
func Open(ref, region string) (StateSource, error) {
	if scheme, _, ok := strings.Cut(ref, "://"); ok {
		f, registered := backends[scheme]
		if !registered {
			return nil, fmt.Errorf("no backend registered for scheme %q (is the %s provider/SDK compiled in?)", scheme, scheme)
		}
		return f(ref, region)
	}
	return NewLocalSource(ref), nil
}

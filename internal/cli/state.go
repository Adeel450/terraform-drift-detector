package cli

import (
	"github.com/adeel450/terraform-drift-detector/internal/tfstate"
)

// stateSourceFor builds a StateSource from a reference. A "s3://bucket/key"
// reference uses the S3 backend (available when cloud providers are compiled
// in); anything else is treated as a local file path.
func stateSourceFor(ref, region string) (tfstate.StateSource, error) {
	return tfstate.Open(ref, region)
}

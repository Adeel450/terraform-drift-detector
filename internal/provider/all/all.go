// Package all blank-imports every provider so their init() functions register
// them (and any state backends they supply). Import this once from main.
package all

import (
	_ "github.com/adeel450/terraform-drift-detector/internal/provider/aws"
	_ "github.com/adeel450/terraform-drift-detector/internal/provider/azure"
	_ "github.com/adeel450/terraform-drift-detector/internal/provider/gcp"
	_ "github.com/adeel450/terraform-drift-detector/internal/provider/mock"
)

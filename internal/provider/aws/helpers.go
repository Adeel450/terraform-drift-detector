package aws

import "github.com/adeel450/terraform-drift-detector/internal/tfstate"

// tfInstance wraps a state instance with convenience accessors used by the AWS
// resource mappers.
type tfInstance struct{ tfstate.Instance }

func (i tfInstance) str(key string) string   { return tfstate.AttrString(i.Attributes, key) }
func (i tfInstance) name() string            { return i.Name }
func (i tfInstance) tags() map[string]string { return tfstate.AttrTags(i.Attributes, "tags") }

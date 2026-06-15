package mock

import (
	"context"
	"testing"

	"github.com/adeel450/terraform-drift-detector/internal/tfstate"
)

const fixtureJSON = `{
  "types": ["aws_instance","aws_security_group"],
  "resources": [
    {"type":"aws_instance","id":"i-1","attributes":{"instance_type":"t3.large"},"tags":{"env":"staging"}}
  ]
}`

func TestMock_MappersCoverDeclaredTypes(t *testing.T) {
	p, err := FromBytes([]byte(fixtureJSON))
	if err != nil {
		t.Fatal(err)
	}
	types := map[string]bool{}
	for _, m := range p.Mappers() {
		types[m.TerraformType()] = true
	}
	if !types["aws_instance"] || !types["aws_security_group"] {
		t.Fatalf("expected mappers for both declared types, got %v", types)
	}
}

func TestMock_FetchActual(t *testing.T) {
	p, err := FromBytes([]byte(fixtureJSON))
	if err != nil {
		t.Fatal(err)
	}
	var im *mapper
	for _, m := range p.Mappers() {
		if m.TerraformType() == "aws_instance" {
			im = m.(*mapper)
		}
	}
	if im == nil {
		t.Fatal("no aws_instance mapper")
	}

	res, found, err := im.FetchActual(context.Background(), "i-1")
	if err != nil || !found {
		t.Fatalf("expected found, got found=%v err=%v", found, err)
	}
	if res.Attributes["instance_type"] != "t3.large" || res.Tags["env"] != "staging" {
		t.Errorf("unexpected actual: %+v", res)
	}

	if _, found, _ := im.FetchActual(context.Background(), "sg-missing"); found {
		t.Error("expected missing id to be not found")
	}

	// FromState reads the same attribute keys declared in the fixtureJSON entry.
	exp, _ := im.FromState(tfstate.Instance{
		Type: "aws_instance", Name: "web",
		Attributes: map[string]any{"id": "i-1", "instance_type": "t3.small", "tags": map[string]any{"env": "prod"}},
	})
	if exp.Attributes["instance_type"] != "t3.small" {
		t.Errorf("expected state instance_type, got %+v", exp.Attributes)
	}
}

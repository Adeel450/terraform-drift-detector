package diff

import (
	"testing"

	"github.com/adeel450/terraform-drift-detector/internal/model"
)

func res(attrs, tags map[string]string) model.Resource {
	return model.Resource{
		Provider: "mock", Type: "aws_instance", ID: "i-1", Name: "web",
		Attributes: attrs, Tags: tags,
	}
}

func TestCompare_Deleted(t *testing.T) {
	exp := res(map[string]string{"instance_type": "t3.small"}, nil)
	items := Compare(exp, model.Resource{}, false)
	if len(items) != 1 || items[0].Kind != model.DriftDeleted {
		t.Fatalf("expected one deleted item, got %+v", items)
	}
}

func TestCompare_ModifiedAttribute(t *testing.T) {
	exp := res(map[string]string{"instance_type": "t3.small", "ami": "ami-1"}, nil)
	act := res(map[string]string{"instance_type": "t3.large", "ami": "ami-1"}, nil)
	items := Compare(exp, act, true)
	if len(items) != 1 {
		t.Fatalf("expected 1 modified item, got %d: %+v", len(items), items)
	}
	it := items[0]
	if it.Kind != model.DriftModified || it.Field != "instance_type" || it.Expected != "t3.small" || it.Actual != "t3.large" {
		t.Fatalf("unexpected item: %+v", it)
	}
}

func TestCompare_Tags_AddRemoveChange(t *testing.T) {
	exp := res(nil, map[string]string{"env": "prod", "team": "platform"})
	act := res(nil, map[string]string{"env": "staging", "owner": "ops"})
	items := Compare(exp, act, true)

	kinds := map[string]model.DriftItem{}
	for _, it := range items {
		if it.Kind != model.DriftTag {
			t.Fatalf("expected only tag drift, got %s", it.Kind)
		}
		kinds[it.Field] = it
	}
	if len(kinds) != 3 {
		t.Fatalf("expected 3 tag changes (env changed, team removed, owner added), got %d: %+v", len(kinds), items)
	}
	if kinds["env"].Expected != "prod" || kinds["env"].Actual != "staging" {
		t.Errorf("env change wrong: %+v", kinds["env"])
	}
	if kinds["team"].Actual != "" {
		t.Errorf("team should be removed (empty actual): %+v", kinds["team"])
	}
	if kinds["owner"].Expected != "" || kinds["owner"].Actual != "ops" {
		t.Errorf("owner should be added: %+v", kinds["owner"])
	}
}

func TestCompare_NoDrift(t *testing.T) {
	exp := res(map[string]string{"instance_type": "t3.small"}, map[string]string{"env": "prod"})
	act := res(map[string]string{"instance_type": "t3.small"}, map[string]string{"env": "prod"})
	if items := Compare(exp, act, true); len(items) != 0 {
		t.Fatalf("expected no drift, got %+v", items)
	}
}

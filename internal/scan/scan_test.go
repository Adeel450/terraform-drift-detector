package scan

import (
	"context"
	"testing"

	"github.com/adeel450/terraform-drift-detector/internal/provider/mock"
)

// staticSource is an in-memory StateSource for tests.
type staticSource struct{ data []byte }

func (s staticSource) Read(context.Context) ([]byte, string, error) {
	return s.data, "memory://test", nil
}

const stateJSON = `{
  "version": 4,
  "resources": [
    {"mode":"managed","type":"aws_instance","name":"web","instances":[
      {"attributes":{"id":"i-1","instance_type":"t3.small","tags":{"env":"prod","team":"platform"}}}]},
    {"mode":"managed","type":"aws_security_group","name":"gone","instances":[
      {"attributes":{"id":"sg-1","tags":{"env":"prod"}}}]},
    {"mode":"data","type":"aws_caller_identity","name":"cur","instances":[
      {"attributes":{"id":"123"}}]}
  ]
}`

const fixtureJSON = `{
  "types": ["aws_instance","aws_security_group"],
  "resources": [
    {"type":"aws_instance","id":"i-1","attributes":{"instance_type":"t3.large"},"tags":{"env":"staging","owner":"ops"}}
  ]
}`

func TestScan_EndToEnd(t *testing.T) {
	p, err := mock.FromBytes([]byte(fixtureJSON))
	if err != nil {
		t.Fatal(err)
	}
	rep, err := Scan(context.Background(), staticSource{[]byte(stateJSON)}, p, Options{})
	if err != nil {
		t.Fatal(err)
	}

	// Data resource is skipped; 2 managed resources checked.
	if rep.Summary.ResourcesChecked != 2 {
		t.Errorf("checked = %d, want 2", rep.Summary.ResourcesChecked)
	}
	// i-1: instance_type modified, env changed, team removed, owner added = 1 modified + 3 tag.
	// sg-1: deleted.
	if rep.Summary.Deleted != 1 {
		t.Errorf("deleted = %d, want 1", rep.Summary.Deleted)
	}
	if rep.Summary.Modified != 1 {
		t.Errorf("modified = %d, want 1", rep.Summary.Modified)
	}
	if rep.Summary.TagChanges != 3 {
		t.Errorf("tag changes = %d, want 3", rep.Summary.TagChanges)
	}
	if rep.Summary.Total != 5 {
		t.Errorf("total = %d, want 5", rep.Summary.Total)
	}
	if !rep.HasDrift() {
		t.Error("expected HasDrift to be true")
	}
	if rep.Provider != "mock" {
		t.Errorf("provider = %q", rep.Provider)
	}
}

func TestScan_NoDrift(t *testing.T) {
	fixture := `{"types":["aws_instance"],"resources":[
      {"type":"aws_instance","id":"i-1","attributes":{"instance_type":"t3.small"},"tags":{"env":"prod","team":"platform"}}]}`
	state := `{"version":4,"resources":[
      {"mode":"managed","type":"aws_instance","name":"web","instances":[
        {"attributes":{"id":"i-1","instance_type":"t3.small","tags":{"env":"prod","team":"platform"}}}]}]}`
	p, err := mock.FromBytes([]byte(fixture))
	if err != nil {
		t.Fatal(err)
	}
	rep, err := Scan(context.Background(), staticSource{[]byte(state)}, p, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if rep.HasDrift() {
		t.Errorf("expected no drift, got %+v", rep.Items)
	}
}

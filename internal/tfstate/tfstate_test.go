package tfstate

import "testing"

func TestParse_SkipsDataResources(t *testing.T) {
	const s = `{"version":4,"resources":[
      {"mode":"managed","type":"aws_instance","name":"a","instances":[{"attributes":{"id":"i-1"}}]},
      {"mode":"data","type":"aws_ami","name":"b","instances":[{"attributes":{"id":"ami-1"}}]}
    ]}`
	insts, err := Parse([]byte(s))
	if err != nil {
		t.Fatal(err)
	}
	if len(insts) != 1 {
		t.Fatalf("want 1 managed instance, got %d", len(insts))
	}
	if insts[0].Type != "aws_instance" || insts[0].ProviderType != "aws" {
		t.Fatalf("unexpected instance: %+v", insts[0])
	}
}

func TestAttrString(t *testing.T) {
	attrs := map[string]any{"s": "x", "n": float64(42), "b": true, "f": 1.5}
	cases := map[string]string{"s": "x", "n": "42", "b": "true", "f": "1.5", "missing": ""}
	for k, want := range cases {
		if got := AttrString(attrs, k); got != want {
			t.Errorf("AttrString(%q) = %q, want %q", k, got, want)
		}
	}
}

func TestAttrTags(t *testing.T) {
	attrs := map[string]any{"tags": map[string]any{"env": "prod", "n": float64(1)}}
	tags := AttrTags(attrs, "tags")
	if tags["env"] != "prod" || tags["n"] != "1" {
		t.Fatalf("unexpected tags: %+v", tags)
	}
	if got := AttrTags(attrs, "missing"); len(got) != 0 {
		t.Fatalf("expected empty map, got %+v", got)
	}
}

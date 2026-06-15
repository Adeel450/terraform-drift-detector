// Package diff compares an expected resource (from Terraform state) against its
// actual counterpart (from a cloud API) and emits drift findings.
package diff

import (
	"fmt"
	"sort"

	"github.com/adeel450/terraform-drift-detector/internal/model"
)

// Compare produces drift items for a single resource. found reports whether the
// actual resource exists in the cloud; when false, the resource is reported as
// deleted and attributes/tags are not compared.
func Compare(expected model.Resource, actual model.Resource, found bool) []model.DriftItem {
	if !found {
		return []model.DriftItem{{
			Key:     expected.Key(),
			Name:    expected.Name,
			Kind:    model.DriftDeleted,
			Message: fmt.Sprintf("resource %s is in Terraform state but was not found in the cloud", expected.Key()),
		}}
	}

	var items []model.DriftItem
	items = append(items, compareAttributes(expected, actual)...)
	items = append(items, compareTags(expected, actual)...)
	return items
}

// compareAttributes reports DriftModified for each differing attribute. Only
// keys present on the expected side are compared (the mapper defines the
// comparable set via FromState).
func compareAttributes(expected, actual model.Resource) []model.DriftItem {
	var items []model.DriftItem
	for _, k := range sortedKeys(expected.Attributes) {
		ev := expected.Attributes[k]
		av, ok := actual.Attributes[k]
		if ok && ev == av {
			continue
		}
		items = append(items, model.DriftItem{
			Key:      expected.Key(),
			Name:     expected.Name,
			Kind:     model.DriftModified,
			Field:    k,
			Expected: ev,
			Actual:   av,
			Message:  fmt.Sprintf("attribute %q changed: expected %q, actual %q", k, ev, av),
		})
	}
	return items
}

// compareTags reports DriftTag for added, removed, or changed tags. The full
// tag maps on both sides are compared so out-of-band additions are caught too.
func compareTags(expected, actual model.Resource) []model.DriftItem {
	var items []model.DriftItem
	seen := map[string]bool{}
	for _, k := range sortedKeys(expected.Tags) {
		seen[k] = true
		ev := expected.Tags[k]
		av, ok := actual.Tags[k]
		switch {
		case !ok:
			items = append(items, tagItem(expected, k, ev, "", fmt.Sprintf("tag %q removed in cloud (expected %q)", k, ev)))
		case ev != av:
			items = append(items, tagItem(expected, k, ev, av, fmt.Sprintf("tag %q changed: expected %q, actual %q", k, ev, av)))
		}
	}
	for _, k := range sortedKeys(actual.Tags) {
		if seen[k] {
			continue
		}
		items = append(items, tagItem(expected, k, "", actual.Tags[k], fmt.Sprintf("tag %q added out-of-band in cloud (value %q)", k, actual.Tags[k])))
	}
	return items
}

func tagItem(expected model.Resource, field, ev, av, msg string) model.DriftItem {
	return model.DriftItem{
		Key:      expected.Key(),
		Name:     expected.Name,
		Kind:     model.DriftTag,
		Field:    field,
		Expected: ev,
		Actual:   av,
		Message:  msg,
	}
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

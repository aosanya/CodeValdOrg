package codevaldorg_test

import (
	"testing"

	codevaldorg "github.com/aosanya/CodeValdOrg"
)

func TestDefaultOrgSchema_TypeCount(t *testing.T) {
	schema := codevaldorg.DefaultOrgSchema()
	if got := len(schema.Types); got != 15 {
		t.Errorf("expected 15 type definitions, got %d", got)
	}
}

func TestDefaultOrgSchema_ImmutableTypes(t *testing.T) {
	schema := codevaldorg.DefaultOrgSchema()
	immutable := map[string]bool{
		"AuthorizationCode": true,
		"AccessToken":       true,
		"RefreshToken":      true,
		"TokenRevocation":   true,
		"AuditEvent":        true,
	}
	for _, td := range schema.Types {
		expected := immutable[td.Name]
		if td.Immutable != expected {
			t.Errorf("type %q: Immutable=%v, expected %v", td.Name, td.Immutable, expected)
		}
	}
}

func TestDefaultOrgSchema_StorageCollections(t *testing.T) {
	schema := codevaldorg.DefaultOrgSchema()

	collectionCounts := map[string]int{}
	for _, td := range schema.Types {
		collectionCounts[td.StorageCollection]++
	}

	cases := []struct {
		collection string
		want       int
	}{
		{"org_entities", 7},
		{"org_oauth_clients", 3},
		{"org_oauth_artifacts", 4},
		{"org_audit_events", 1},
	}

	for _, c := range cases {
		got := collectionCounts[c.collection]
		if got != c.want {
			t.Errorf("collection %q: got %d types, want %d", c.collection, got, c.want)
		}
	}
}

func TestDefaultOrgSchema_ID(t *testing.T) {
	schema := codevaldorg.DefaultOrgSchema()
	if schema.ID != "org-schema-v1" {
		t.Errorf("expected schema ID org-schema-v1, got %q", schema.ID)
	}
	if schema.Version != 1 {
		t.Errorf("expected version 1, got %d", schema.Version)
	}
	if schema.Tag != "v1" {
		t.Errorf("expected tag v1, got %q", schema.Tag)
	}
}

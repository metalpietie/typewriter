package main

import (
	"testing"
)

func TestRegionKindAndCascade(t *testing.T) {
	if !kinds["region"] {
		t.Fatal("region is not a known kind")
	}
	found := false
	for _, c := range cascades["map"] {
		if c[0] == "region" && c[1] == "map" {
			found = true
		}
	}
	if !found {
		t.Error("deleting a map must delete its regions")
	}
}

func TestDraftKind(t *testing.T) {
	if !kinds["draft"] {
		t.Fatal("draft is not a known kind")
	}
}

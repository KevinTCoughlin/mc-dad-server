package parkour

import (
	"testing"
)

func TestDefaultMaps(t *testing.T) {
	maps := DefaultMaps()
	if len(maps) != 5 {
		t.Fatalf("expected 5 maps, got %d", len(maps))
	}

	expected := []string{
		"parkour-spiral",
		"parkour-spiral-3",
		"parkour-volcano",
		"parkour-pyramid",
		"parkour-paradise",
	}

	for i, m := range maps {
		if m.Name != expected[i] {
			t.Errorf("map %d: expected name %q, got %q", i, expected[i], m.Name)
		}
		if m.URL == "" {
			t.Errorf("map %d (%s): URL is empty", i, m.Name)
		}
	}
}

func TestParkourWorldYML(t *testing.T) {
	if ParkourWorldYML == "" {
		t.Fatal("ParkourWorldYML should not be empty")
	}
}

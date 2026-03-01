package ops_test

import (
	"encoding/json"
	"testing"

	ops "github.com/machinefabric/ops-go"
)

// TEST024: Build a flat ListingOutline with depth-0 entries and verify max_depth, levels, and flatten count
func Test024_simple_flat_outline(t *testing.T) {
	p := func(s string) *string { return &s }

	outline := ops.NewListingOutline()
	outline.Entries = []ops.OutlineEntry{
		ops.NewOutlineEntry("Introduction", p("1"), 0),
		ops.NewOutlineEntry("Chapter 1: Getting Started", p("5"), 0),
		ops.NewOutlineEntry("Chapter 2: Advanced Topics", p("15"), 0),
		ops.NewOutlineEntry("Conclusion", p("25"), 0),
	}

	if outline.MaxDepth() != 0 {
		t.Errorf("expected max_depth=0, got %d", outline.MaxDepth())
	}
	if len(outline.EntriesAtLevel(0)) != 4 {
		t.Errorf("expected 4 entries at level 0, got %d", len(outline.EntriesAtLevel(0)))
	}
	if len(outline.Flatten()) != 4 {
		t.Errorf("expected 4 flattened entries, got %d", len(outline.Flatten()))
	}
}

// TEST025: Build a two-level outline with chapters and sections and verify depth, level counts, and flatten
func Test025_hierarchical_outline(t *testing.T) {
	p := func(s string) *string { return &s }

	chapter1 := ops.NewOutlineEntry("Chapter 1: Basics", p("10"), 0).WithType("chapter")
	chapter1.AddChild(ops.NewOutlineEntry("1.1 Introduction", p("10"), 1))
	chapter1.AddChild(ops.NewOutlineEntry("1.2 Fundamentals", p("15"), 1))

	chapter2 := ops.NewOutlineEntry("Chapter 2: Advanced", p("20"), 0).WithType("chapter")
	chapter2.AddChild(ops.NewOutlineEntry("2.1 Complex Topics", p("20"), 1))

	outline := ops.NewListingOutline()
	outline.Entries = []ops.OutlineEntry{chapter1, chapter2}

	if outline.MaxDepth() != 1 {
		t.Errorf("expected max_depth=1, got %d", outline.MaxDepth())
	}
	if len(outline.EntriesAtLevel(0)) != 2 {
		t.Errorf("expected 2 entries at level 0, got %d", len(outline.EntriesAtLevel(0)))
	}
	if len(outline.EntriesAtLevel(1)) != 3 {
		t.Errorf("expected 3 entries at level 1, got %d", len(outline.EntriesAtLevel(1)))
	}
	if len(outline.Flatten()) != 5 {
		t.Errorf("expected 5 flattened entries (2 chapters + 3 sections), got %d", len(outline.Flatten()))
	}
}

// TEST026: Build a three-level part/chapter/section outline and verify depth and per-level entry counts
func Test026_complex_part_based_outline(t *testing.T) {
	p := func(s string) *string { return &s }

	part1 := ops.NewOutlineEntry("Part I: Foundations", p("1"), 0).WithType("part")

	chapter1 := ops.NewOutlineEntry("Chapter 1: Introduction", p("3"), 1).WithType("chapter")
	chapter1.AddChild(ops.NewOutlineEntry("1.1 Overview", p("3"), 2))
	chapter1.AddChild(ops.NewOutlineEntry("1.2 Scope", p("5"), 2))

	chapter2 := ops.NewOutlineEntry("Chapter 2: Background", p("8"), 1).WithType("chapter")

	part1.AddChild(chapter1)
	part1.AddChild(chapter2)

	part2 := ops.NewOutlineEntry("Part II: Applications", p("15"), 0).WithType("part")

	outline := ops.NewListingOutline()
	outline.Entries = []ops.OutlineEntry{part1, part2}

	if outline.MaxDepth() != 2 {
		t.Errorf("expected max_depth=2, got %d", outline.MaxDepth())
	}
	if len(outline.EntriesAtLevel(0)) != 2 {
		t.Errorf("expected 2 entries at level 0 (parts), got %d", len(outline.EntriesAtLevel(0)))
	}
	if len(outline.EntriesAtLevel(1)) != 2 {
		t.Errorf("expected 2 entries at level 1 (chapters), got %d", len(outline.EntriesAtLevel(1)))
	}
	if len(outline.EntriesAtLevel(2)) != 2 {
		t.Errorf("expected 2 entries at level 2 (sections), got %d", len(outline.EntriesAtLevel(2)))
	}
	if len(outline.Flatten()) != 6 {
		t.Errorf("expected 6 flattened entries (2+2+2), got %d", len(outline.Flatten()))
	}
}

// TEST027: Flatten a nested outline and verify each entry's path reflects its ancestry correctly
func Test027_flatten_preserves_hierarchy(t *testing.T) {
	p := func(s string) *string { return &s }

	part := ops.NewOutlineEntry("Part I", p("1"), 0)
	chapter := ops.NewOutlineEntry("Chapter 1", p("3"), 1)
	chapter.AddChild(ops.NewOutlineEntry("Section 1.1", p("3"), 2))
	part.AddChild(chapter)

	outline := ops.NewListingOutline()
	outline.Entries = []ops.OutlineEntry{part}

	flat := outline.Flatten()
	if len(flat) != 3 {
		t.Fatalf("expected 3 flattened entries, got %d", len(flat))
	}

	if len(flat[0].Path) != 1 || flat[0].Path[0] != "Part I" {
		t.Errorf("expected path=['Part I'], got %v", flat[0].Path)
	}
	if len(flat[1].Path) != 2 || flat[1].Path[0] != "Part I" || flat[1].Path[1] != "Chapter 1" {
		t.Errorf("expected path=['Part I','Chapter 1'], got %v", flat[1].Path)
	}
	if len(flat[2].Path) != 3 || flat[2].Path[2] != "Section 1.1" {
		t.Errorf("expected path=['Part I','Chapter 1','Section 1.1'], got %v", flat[2].Path)
	}
}

// TEST028: Call GenerateOutlineSchema and verify the returned JSON contains all required definitions
func Test028_schema_generation(t *testing.T) {
	schema := ops.GenerateOutlineSchema()

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(schema, &obj); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}

	propsRaw, ok := obj["properties"]
	if !ok {
		t.Fatal("expected 'properties' key in schema")
	}
	var props map[string]json.RawMessage
	if err := json.Unmarshal(propsRaw, &props); err != nil {
		t.Fatalf("properties is not valid JSON: %v", err)
	}
	if _, ok := props["entries"]; !ok {
		t.Error("expected 'entries' in properties")
	}

	defsRaw, ok := obj["definitions"]
	if !ok {
		t.Fatal("expected 'definitions' key in schema")
	}
	var defs map[string]json.RawMessage
	if err := json.Unmarshal(defsRaw, &defs); err != nil {
		t.Fatalf("definitions is not valid JSON: %v", err)
	}
	if _, ok := defs["OutlineEntry"]; !ok {
		t.Error("expected 'OutlineEntry' in definitions")
	}
	if _, ok := defs["OutlineMetadata"]; !ok {
		t.Error("expected 'OutlineMetadata' in definitions")
	}
}

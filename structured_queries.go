package ops

import "encoding/json"

// OutlineEntry represents a single hierarchical table of contents entry.
type OutlineEntry struct {
	Title     string         `json:"title"`
	Page      *string        `json:"page,omitempty"`
	Level     uint8          `json:"level"`
	EntryType *string        `json:"entry_type,omitempty"`
	Children  []OutlineEntry `json:"children"`
}

// NewOutlineEntry creates an OutlineEntry with the given title, page, and level.
func NewOutlineEntry(title string, page *string, level uint8) OutlineEntry {
	return OutlineEntry{Title: title, Page: page, Level: level, Children: []OutlineEntry{}}
}

// WithType returns a copy of this entry with the given type set.
func (e OutlineEntry) WithType(entryType string) OutlineEntry {
	e.EntryType = &entryType
	return e
}

// WithChildren returns a copy of this entry with the given children set.
func (e OutlineEntry) WithChildren(children []OutlineEntry) OutlineEntry {
	e.Children = children
	return e
}

// AddChild appends a child entry.
func (e *OutlineEntry) AddChild(child OutlineEntry) {
	e.Children = append(e.Children, child)
}

// Flatten returns all entries in this subtree as a flat list with path information.
func (e *OutlineEntry) Flatten() []FlatOutlineEntry {
	var result []FlatOutlineEntry
	e.flattenRecursive(&result, nil)
	return result
}

func (e *OutlineEntry) flattenRecursive(result *[]FlatOutlineEntry, path []string) {
	path = append(path, e.Title)
	pathCopy := make([]string, len(path))
	copy(pathCopy, path)
	*result = append(*result, FlatOutlineEntry{
		Title:     e.Title,
		Page:      e.Page,
		Level:     e.Level,
		EntryType: e.EntryType,
		Path:      pathCopy,
	})
	for i := range e.Children {
		e.Children[i].flattenRecursive(result, path)
	}
}

// FlatOutlineEntry is an OutlineEntry with its full hierarchical path recorded.
type FlatOutlineEntry struct {
	Title     string  `json:"title"`
	Page      *string `json:"page,omitempty"`
	Level     uint8   `json:"level"`
	EntryType *string `json:"entry_type,omitempty"`
	Path      []string `json:"path"`
}

// ListingOutline represents a complete hierarchical table of contents.
type ListingOutline struct {
	DocumentTitle *string         `json:"document_title,omitempty"`
	Entries       []OutlineEntry  `json:"entries"`
	Confidence    float64         `json:"confidence"`
	Metadata      OutlineMetadata `json:"metadata"`
}

// NewListingOutline creates an empty ListingOutline.
func NewListingOutline() *ListingOutline {
	return &ListingOutline{Entries: []OutlineEntry{}}
}

// Flatten returns all entries in the outline as a flat list.
func (l *ListingOutline) Flatten() []FlatOutlineEntry {
	var result []FlatOutlineEntry
	for i := range l.Entries {
		result = append(result, l.Entries[i].Flatten()...)
	}
	return result
}

// EntriesAtLevel returns all entries at the given hierarchical level.
func (l *ListingOutline) EntriesAtLevel(level uint8) []*OutlineEntry {
	var result []*OutlineEntry
	collectAtLevel(l.Entries, level, &result)
	return result
}

func collectAtLevel(entries []OutlineEntry, targetLevel uint8, result *[]*OutlineEntry) {
	for i := range entries {
		if entries[i].Level == targetLevel {
			*result = append(*result, &entries[i])
		}
		collectAtLevel(entries[i].Children, targetLevel, result)
	}
}

// MaxDepth returns the maximum hierarchical depth of entries.
func (l *ListingOutline) MaxDepth() uint8 {
	return maxDepthRecursive(l.Entries)
}

func maxDepthRecursive(entries []OutlineEntry) uint8 {
	var max uint8
	for _, e := range entries {
		d := e.Level
		if len(e.Children) > 0 {
			childMax := maxDepthRecursive(e.Children)
			if childMax > d {
				d = childMax
			}
		}
		if d > max {
			max = d
		}
	}
	return max
}

// OutlineMetadata contains metadata about the outline structure.
type OutlineMetadata struct {
	NumberingStyle *string `json:"numbering_style,omitempty"`
	HasLeaders     bool    `json:"has_leaders"`
	PageStyle      *string `json:"page_style,omitempty"`
	TotalEntries   int     `json:"total_entries"`
	Levels         uint8   `json:"levels"`
	StructureType  *string `json:"structure_type,omitempty"`
}

// GenerateOutlineSchema returns the JSON Schema for ListingOutline.
func GenerateOutlineSchema() json.RawMessage {
	schema := map[string]any{
		"$schema":     "http://json-schema.org/draft-07/schema#",
		"title":       "Table of Contents",
		"description": "Hierarchical table of contents structure",
		"type":        "object",
		"properties": map[string]any{
			"document_title": map[string]any{
				"type":        []string{"string", "null"},
				"description": "Title of the document (optional)",
			},
			"entries": map[string]any{
				"type":        "array",
				"description": "Main table of contents entries",
				"items":       map[string]any{"$ref": "#/definitions/OutlineEntry"},
			},
			"confidence": map[string]any{
				"type":        "number",
				"minimum":     0.0,
				"maximum":     1.0,
				"description": "Confidence level of the extraction (0.0 - 1.0)",
			},
			"metadata": map[string]any{"$ref": "#/definitions/OutlineMetadata"},
		},
		"required": []string{"entries", "confidence"},
		"definitions": map[string]any{
			"OutlineEntry": map[string]any{
				"type":        "object",
				"description": "A single table of contents entry with optional hierarchy",
				"properties": map[string]any{
					"title":      map[string]any{"type": "string"},
					"page":       map[string]any{"type": []string{"string", "null"}},
					"level":      map[string]any{"type": "integer", "minimum": 0, "maximum": 10},
					"entry_type": map[string]any{"type": []string{"string", "null"}},
					"children": map[string]any{
						"type":  "array",
						"items": map[string]any{"$ref": "#/definitions/OutlineEntry"},
					},
				},
				"required": []string{"title", "level"},
			},
			"OutlineMetadata": map[string]any{
				"type":        "object",
				"description": "Metadata about the table of contents structure",
				"properties": map[string]any{
					"numbering_style": map[string]any{"type": []string{"string", "null"}},
					"has_leaders":     map[string]any{"type": "boolean"},
					"page_style":      map[string]any{"type": []string{"string", "null"}},
					"total_entries":   map[string]any{"type": "integer", "minimum": 0},
					"levels":          map[string]any{"type": "integer", "minimum": 1, "maximum": 10},
					"structure_type":  map[string]any{"type": []string{"string", "null"}},
				},
				"required": []string{"has_leaders", "total_entries", "levels"},
			},
		},
	}
	data, _ := json.Marshal(schema)
	return data
}

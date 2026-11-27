package client

import "testing"

func TestCodexDisplayHelpers(t *testing.T) {
	cases := []struct {
		name string
		e    CodexEntry
		want CodexEntryDisplay
	}{
		{"type wins", CodexEntry{Type: "character", Category: "lore"}, CodexEntryDisplay{Type: "character", Details: "-", Aliases: "-"}},
		{"category fallback", CodexEntry{Category: "lore"}, CodexEntryDisplay{Type: "lore", Details: "-", Aliases: "-"}},
		{"defaults", CodexEntry{}, CodexEntryDisplay{Type: "lore", Details: "-", Aliases: "-"}},
		{"details wins", CodexEntry{Details: "D", Content: "C"}, CodexEntryDisplay{Type: "lore", Details: "D", Aliases: "-"}},
		{"content fallback", CodexEntry{Content: "C"}, CodexEntryDisplay{Type: "lore", Details: "C", Aliases: "-"}},
		{"aliases", CodexEntry{Aliases: strptr("A")}, CodexEntryDisplay{Type: "lore", Details: "-", Aliases: "A"}},
		{"whitespace aliases", CodexEntry{Aliases: strptr("  \n ")}, CodexEntryDisplay{Type: "lore", Details: "-", Aliases: "-"}},
		{"empty aliases", CodexEntry{Aliases: strptr("")}, CodexEntryDisplay{Type: "lore", Details: "-", Aliases: "-"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.e.Display()
			if got != tc.want {
				t.Fatalf("Display() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestCodexNormalize(t *testing.T) {
	cases := []struct {
		name string
		e    CodexEntry
	}{
		{"type to category", CodexEntry{Type: "character"}},
		{"category to type", CodexEntry{Category: "place"}},
		{"details to content", CodexEntry{Details: "D"}},
		{"content to details", CodexEntry{Content: "C"}},
		{"both empty", CodexEntry{}},
		{"both set", CodexEntry{Type: "character", Category: "place", Details: "D", Content: "C"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := tc.e
			e.normalize()
			wantType := tc.e.Type
			if wantType == "" {
				wantType = tc.e.Category
			}
			if wantType == "" {
				wantType = "lore"
			}
			wantCategory := tc.e.Category
			if wantCategory == "" {
				wantCategory = wantType
			}
			if e.Type != wantType {
				t.Fatalf("normalize: type %q, want %q", e.Type, wantType)
			}
			if e.Category != wantCategory {
				t.Fatalf("normalize: category %q, want %q", e.Category, wantCategory)
			}
			wantDetails := tc.e.Details
			if wantDetails == "" {
				wantDetails = tc.e.Content
			}
			wantContent := tc.e.Content
			if wantContent == "" {
				wantContent = wantDetails
			}
			if e.Details != wantDetails {
				t.Fatalf("normalize: details %q, want %q", e.Details, wantDetails)
			}
			if e.Content != wantContent {
				t.Fatalf("normalize: content %q, want %q", e.Content, wantContent)
			}
		})
	}
}

func TestDecodeCodexEntryVariants(t *testing.T) {
	e, err := decodeCodexEntry([]byte(`{"data":{"id":9,"name":"X"}}`))
	if err != nil || e.ID != 9 || e.Name != "X" {
		t.Fatalf("envelope decode = %+v, err %v", e, err)
	}
	e, err = decodeCodexEntry([]byte(`{"id":4,"name":"Bare"}`))
	if err != nil || e.ID != 4 || e.Name != "Bare" {
		t.Fatalf("bare decode = %+v, err %v", e, err)
	}
	e, err = decodeCodexEntry([]byte(`{"data":{"id":0}}`))
	if err != nil || e.ID != 0 {
		t.Fatalf("zero-id envelope = %+v, err %v", e, err)
	}
	if _, err := decodeCodexEntry([]byte(`{broken`)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

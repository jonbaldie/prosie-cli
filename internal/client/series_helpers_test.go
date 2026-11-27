package client

import "testing"

func TestSeriesDisplayHelpers(t *testing.T) {
	cases := []struct {
		name string
		s    Series
		want SeriesDisplay
	}{
		{"id only", Series{ID: 7}, SeriesDisplay{Title: "Series 7", Description: "-"}},
		{"title wins", Series{ID: 7, Title: "T", Name: "N"}, SeriesDisplay{Title: "T", Description: "-"}},
		{"name fallback", Series{ID: 7, Name: "N"}, SeriesDisplay{Title: "N", Description: "-"}},
		{"description", Series{ID: 7, Description: strptr("D")}, SeriesDisplay{Title: "Series 7", Description: "D"}},
		{"whitespace description", Series{ID: 7, Description: strptr("  \t ")}, SeriesDisplay{Title: "Series 7", Description: "-"}},
		{"empty description", Series{ID: 7, Description: strptr("")}, SeriesDisplay{Title: "Series 7", Description: "-"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.s.Display()
			if got != tc.want {
				t.Fatalf("Display() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestSeriesNormalizeTitles(t *testing.T) {
	cases := []struct {
		name string
		s    Series
	}{
		{"name to title", Series{Name: "N"}},
		{"title to name", Series{Title: "T"}},
		{"both empty", Series{}},
		{"both set", Series{Name: "N", Title: "T"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := tc.s
			s.normalize()
			if tc.s.Name != "" || tc.s.Title != "" {
				if tc.s.Name == "" {
					if s.Title != tc.s.Title {
						t.Fatalf("normalize: title %q, want %q", s.Title, tc.s.Title)
					}
				}
				if tc.s.Title == "" {
					if s.Name != tc.s.Name {
						t.Fatalf("normalize: name %q, want %q", s.Name, tc.s.Name)
					}
				}
			}
		})
	}
}

func TestSeriesNormalizeBooks(t *testing.T) {
	// Books empty, stories present: books = stories.
	book := Book{ID: 5, Title: "B"}
	s := Series{Stories: []Book{book}}
	s.normalize()
	if len(s.Books) != 1 || s.Books[0].ID != 5 {
		t.Fatalf("expected books from stories, got %+v", s.Books)
	}
	// Stories empty, books present: stories = books.
	s = Series{Books: []Book{book}}
	s.normalize()
	if len(s.Stories) != 1 || s.Stories[0].ID != 5 {
		t.Fatalf("expected stories from books, got %+v", s.Stories)
	}
	// Both present: both kept.
	s = Series{Books: []Book{{ID: 1}}, Stories: []Book{{ID: 2}}}
	s.normalize()
	if len(s.Books) != 1 || len(s.Stories) != 1 {
		t.Fatalf("expected both kept, got books=%+v stories=%+v", s.Books, s.Stories)
	}
	// Neither present: both stay empty.
	s = Series{}
	s.normalize()
	if len(s.Books) != 0 || len(s.Stories) != 0 {
		t.Fatalf("expected both empty, got %+v %+v", s.Books, s.Stories)
	}
}

func TestDecodeSeriesVariants(t *testing.T) {
	s, err := decodeSeries([]byte(`{"data":{"id":9,"name":"X"}}`))
	if err != nil || s.ID != 9 || s.Name != "X" {
		t.Fatalf("envelope decode = %+v, err %v", s, err)
	}
	s, err = decodeSeries([]byte(`{"id":4,"name":"Bare"}`))
	if err != nil || s.ID != 4 || s.Name != "Bare" {
		t.Fatalf("bare decode = %+v, err %v", s, err)
	}
	s, err = decodeSeries([]byte(`{"data":{"id":0}}`))
	if err != nil || s.ID != 0 {
		t.Fatalf("zero-id envelope = %+v, err %v", s, err)
	}
	if _, err := decodeSeries([]byte(`{broken`)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestNeedsSeriesBooks(t *testing.T) {
	if !needsSeriesBooks([]Series{{ID: 1}}) {
		t.Fatal("expected true for series without books")
	}
	if needsSeriesBooks([]Series{{ID: 1, Books: []Book{{ID: 2}}}}) {
		t.Fatal("expected false when books present")
	}
	if needsSeriesBooks(nil) {
		t.Fatal("expected false for empty list")
	}
	if !needsSeriesBooks([]Series{{ID: 1, Books: []Book{{ID: 2}}}, {ID: 3}}) {
		t.Fatal("expected true when any series lacks books")
	}
}

func TestSeriesBookIndexMembers(t *testing.T) {
	b1 := Book{ID: 10, SeriesID: intptr(1)}
	b2 := Book{ID: 11, SeriesID: intptr(1)}
	b3 := Book{ID: 12, SeriesID: intptr(2)}
	index := newSeriesBookIndex([]Book{b1, b2, b3})
	// Without StoryIDs: members are books of the series.
	members := index.members(Series{ID: 1})
	if len(members) != 2 {
		t.Fatalf("expected 2 members without story IDs, got %+v", members)
	}
	// With StoryIDs: picks by ID from the index.
	members = index.members(Series{ID: 1, StoryIDs: []int{12, 999}})
	if len(members) != 1 || members[0].ID != 12 {
		t.Fatalf("expected only book 12, got %+v", members)
	}
	// Existing books are kept as members.
	members = index.members(Series{ID: 1, Books: []Book{{ID: 99}}})
	if len(members) != 3 || members[0].ID != 99 || members[1].ID != 10 || members[2].ID != 11 {
		t.Fatalf("expected existing book plus series books, got %+v", members)
	}
	// Series with no books at all.
	members = index.members(Series{ID: 77})
	if len(members) != 0 {
		t.Fatalf("expected no members, got %+v", members)
	}
}

func intptr(n int) *int { return &n }

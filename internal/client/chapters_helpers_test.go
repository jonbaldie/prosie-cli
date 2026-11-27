package client

import "testing"

func TestChapterDisplayHelpers(t *testing.T) {
	cases := []struct {
		name string
		c    Chapter
		want ChapterDisplay
	}{
		{"title wins", Chapter{Order: 2, Title: "T", Name: "N"}, ChapterDisplay{Title: "T", Summary: "-"}},
		{"name fallback", Chapter{Order: 2, Name: "N"}, ChapterDisplay{Title: "N", Summary: "-"}},
		{"numbered fallback", Chapter{Order: 2}, ChapterDisplay{Title: "Chapter 3", Summary: "-"}},
		{"first chapter", Chapter{Order: 0}, ChapterDisplay{Title: "Chapter 1", Summary: "-"}},
		{"summary", Chapter{Order: 0, Summary: strptr("S")}, ChapterDisplay{Title: "Chapter 1", Summary: "S"}},
		{"empty summary", Chapter{Order: 0, Summary: strptr("")}, ChapterDisplay{Title: "Chapter 1", Summary: "-"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.c.Display()
			if got != tc.want {
				t.Fatalf("Display() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestChapterNormalize(t *testing.T) {
	cases := []struct {
		name string
		c    Chapter
	}{
		{"name to title", Chapter{Name: "N"}},
		{"title to name", Chapter{Title: "T"}},
		{"both", Chapter{Name: "N", Title: "T"}},
		{"neither", Chapter{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := tc.c
			c.normalize()
			switch {
			case tc.c.Title != "" && tc.c.Name == "":
				if c.Title != tc.c.Title || c.Name != tc.c.Title {
					t.Fatalf("normalize: title=%q name=%q, want both %q", c.Title, c.Name, tc.c.Title)
				}
			case tc.c.Name != "" && tc.c.Title == "":
				if c.Title != tc.c.Name || c.Name != tc.c.Name {
					t.Fatalf("normalize: title=%q name=%q, want both %q", c.Title, c.Name, tc.c.Name)
				}
			default:
				if c.Title != tc.c.Title || c.Name != tc.c.Name {
					t.Fatalf("normalize: title=%q name=%q must stay unchanged", c.Title, c.Name)
				}
			}
		})
	}
}

func TestDecodeChapterVariants(t *testing.T) {
	ch, err := decodeChapter([]byte(`{"data":{"id":9,"title":"X"}}`))
	if err != nil || ch.ID != 9 || ch.Title != "X" {
		t.Fatalf("envelope decode = %+v, err %v", ch, err)
	}
	ch, err = decodeChapter([]byte(`{"id":4,"title":"Bare"}`))
	if err != nil || ch.ID != 4 || ch.Title != "Bare" {
		t.Fatalf("bare decode = %+v, err %v", ch, err)
	}
	ch, err = decodeChapter([]byte(`{"data":{"id":0}}`))
	if err != nil || ch.ID != 0 {
		t.Fatalf("zero-id envelope = %+v, err %v", ch, err)
	}
	if _, err := decodeChapter([]byte(`{broken`)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

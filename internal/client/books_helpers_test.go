package client

import "testing"

func strptr(s string) *string { return &s }

func TestBookDisplayHelpers(t *testing.T) {
	cases := []struct {
		name string
		book Book
		want BookDisplay
	}{
		{"all nil", Book{}, BookDisplay{Premise: "-", Lore: "-", Characters: "-"}},
		{"all empty", Book{Premise: strptr(""), StorySoFar: strptr(""), Subtitle: strptr(""), Lore: strptr(""), Characters: strptr("")}, BookDisplay{Premise: "-", Lore: "-", Characters: "-"}},
		{"premise", Book{Premise: strptr("P"), StorySoFar: strptr("S"), Subtitle: strptr("Sub")}, BookDisplay{Premise: "P", Lore: "-", Characters: "-"}},
		{"story so far fallback", Book{StorySoFar: strptr("S"), Subtitle: strptr("Sub")}, BookDisplay{Premise: "S", Lore: "-", Characters: "-"}},
		{"subtitle fallback", Book{Subtitle: strptr("Sub")}, BookDisplay{Premise: "Sub", Lore: "-", Characters: "-"}},
		{"empty premise falls to story so far", Book{Premise: strptr(""), StorySoFar: strptr("S")}, BookDisplay{Premise: "S", Lore: "-", Characters: "-"}},
		{"lore and characters", Book{Lore: strptr("L"), Characters: strptr("C")}, BookDisplay{Premise: "-", Lore: "L", Characters: "C"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.book.Display()
			if got != tc.want {
				t.Fatalf("Display() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestBookNormalize(t *testing.T) {
	cases := []struct {
		name string
		book Book
	}{
		{"premise to story so far", Book{Premise: strptr("P")}},
		{"story so far to premise", Book{StorySoFar: strptr("S")}},
		{"both nil", Book{}},
		{"both set", Book{Premise: strptr("P"), StorySoFar: strptr("S")}},
		{"one empty", Book{Premise: strptr(""), StorySoFar: strptr("S")}},
		{"one empty reverse", Book{Premise: strptr("P"), StorySoFar: strptr("")}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			premise, sofar := tc.book.Premise, tc.book.StorySoFar
			tc.book.normalize()
			switch {
			case premise == nil && sofar == nil:
				if tc.book.Premise != nil || tc.book.StorySoFar != nil {
					t.Fatalf("normalize: both fields must stay nil: %+v", tc.book)
				}
			case premise != nil && sofar == nil:
				if *tc.book.Premise != *premise || *tc.book.StorySoFar != *premise {
					t.Fatalf("normalize: premise=%v story=%v, want both %q", tc.book.Premise, tc.book.StorySoFar, *premise)
				}
			case sofar != nil && premise == nil:
				if *tc.book.Premise != *sofar || *tc.book.StorySoFar != *sofar {
					t.Fatalf("normalize: premise=%v story=%v, want both %q", tc.book.Premise, tc.book.StorySoFar, *sofar)
				}
			default:
				if *tc.book.Premise != *premise || *tc.book.StorySoFar != *sofar {
					t.Fatalf("normalize: both fields must stay unchanged: %+v", tc.book)
				}
			}
		})
	}
}

func TestImportedBookTitle(t *testing.T) {
	cases := []struct {
		filename, title, want string
	}{
		{"document.docx", "Given", "Given"},
		{"document.docx", "", "document"},
		{"/tmp/path/My Book.docx", "", "My Book"},
		{"noext", "", "noext"},
		{".docx", "", "Imported Book"},
		{"archive.tar.gz", "", "archive.tar"},
	}
	for _, tc := range cases {
		if got := importedBookTitle(tc.filename, tc.title); got != tc.want {
			t.Fatalf("importedBookTitle(%q, %q) = %q, want %q", tc.filename, tc.title, got, tc.want)
		}
	}
}

func TestDecodeBookVariants(t *testing.T) {
	// Envelope form.
	book, err := decodeBook([]byte(`{"data":{"id":7,"title":"T"}}`))
	if err != nil || book.ID != 7 || book.Title != "T" {
		t.Fatalf("envelope decode = %+v, err %v", book, err)
	}
	// Bare form.
	book, err = decodeBook([]byte(`{"id":3,"title":"Bare"}`))
	if err != nil || book.ID != 3 || book.Title != "Bare" {
		t.Fatalf("bare decode = %+v, err %v", book, err)
	}
	// Envelope with zero ID: falls through to bare decode.
	book, err = decodeBook([]byte(`{"data":{"id":0}}`))
	if err != nil || book.ID != 0 {
		t.Fatalf("zero-id envelope = %+v, err %v", book, err)
	}
	// Invalid JSON.
	if _, err := decodeBook([]byte(`{broken`)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

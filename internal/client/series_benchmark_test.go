package client

import (
	"fmt"
	"testing"
)

func BenchmarkSeriesMembership(b *testing.B) {
	for _, mode := range []string{"explicit", "fallback"} {
		for _, n := range []int{100, 500, 1000} {
			b.Run(fmt.Sprintf("%s/series=%d/books=%d", mode, n, 10*n), func(b *testing.B) {
				books := make([]Book, 10*n)
				series := make([]Series, n)
				for i := range series {
					series[i].ID = i + 1
					if mode == "explicit" {
						series[i].StoryIDs = []int{i + 1}
					}
				}
				for i := range books {
					books[i].ID = i + 1
					books[i].SeriesID = &series[i%n].ID
				}
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					index := newSeriesBookIndex(books)
					for _, s := range series {
						members := index.members(s)
						want := 10
						if mode == "explicit" {
							want = 1
						}
						if len(members) != want || members[0].ID != s.ID || members[want-1].ID != s.ID+(want-1)*n {
							b.Fatalf("series %d: unexpected members: %v", s.ID, members)
						}
					}
				}
			})
		}
	}
}

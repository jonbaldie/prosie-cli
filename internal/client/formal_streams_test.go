package client

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestFormalStreamsFrame(t *testing.T) {
	for _, input := range []string{"event: delta\ndata: one\n", "event: delta\ndata: one"} {
		var got []string
		err := readEvents(strings.NewReader(input), func(event, data string) { got = append(got, event+":"+data) })
		if err != nil || len(got) == 0 || got[0] != "delta:one" || len(got) > 2 || (len(got) == 2 && got[1] != ":") {
			t.Fatalf("EOF frame: %q %v", got, err)
		}
	}
	for _, example := range []struct{ input, want string }{
		{"event: x\nevent: y\ndata: a", "y:a"},
		{"event: x\ndata: a\nevent:", ":a"},
		{"data:\ndata: a", ":\na"},
	} {
		var frames []string
		err := readEvents(strings.NewReader(example.input), func(event, data string) { frames = append(frames, event+":"+data) })
		if err != nil || strings.Join(frames, "|") != example.want {
			t.Fatalf("frame input %q: %q %v", example.input, frames, err)
		}
	}
	var got []string
	err := readEvents(strings.NewReader("event: x\ndata: a\ndata:  b\n\nevent: y\ndata: c"), func(e, d string) { got = append(got, e+":"+d) })
	if err != nil || strings.Join(got, "|") != "x:a\n b|y:c" {
		t.Fatalf("frames: %q %v", got, err)
	}
	got = nil
	err = readEvents(strings.NewReader("event: x\ndata: a\n\r"), func(e, d string) { got = append(got, e+":"+d) })
	if err != nil || strings.Join(got, "|") != "x:a|:" {
		t.Fatalf("blank unterminated EOF line: %q %v", got, err)
	}
	got = nil
	err = readEvents(&formalStreamsBadReader{}, func(e, d string) { got = append(got, e+":"+d) })
	if err == nil || len(got) != 0 {
		t.Fatalf("read error: %q %v", got, err)
	}
}

type formalStreamsBadReader struct{}

func (*formalStreamsBadReader) Read(p []byte) (int, error) {
	return copy(p, "data: lost"), errors.New("read failed")
}

func TestFormalStreamsResult(t *testing.T) {
	type result struct {
		N int `json:"n"`
	}
	var tokens []string
	input := "event: error\ndata: bad\n\nevent: done\ndata: {\"n\":1}\n\nevent: delta\ndata: {\"delta\":\"late\"}\n\n"
	got, err := streamSSE[result](context.Background(), strings.NewReader(input), func(s string) { tokens = append(tokens, s) })
	if err == nil || got != nil || strings.Join(tokens, "") != "late" {
		t.Fatalf("retained error and late token: %#v %v %q", got, err, tokens)
	}
	got, err = streamSSE[result](context.Background(), strings.NewReader("event: done\ndata: {\"n\":1}\n\nevent: done\ndata: {\"n\":2}\n\n"), nil)
	if err != nil || got == nil || got.N != 2 {
		t.Fatalf("last done: %#v %v", got, err)
	}
	got, err = streamSSE[result](context.Background(), strings.NewReader("event: done\ndata: {\"n\":2}\n\nevent: done\ndata: bad\n\nevent: done\ndata: null\n\n"), nil)
	if err != nil || got == nil || got.N != 0 {
		t.Fatalf("null replaces prior done with zero: %#v %v", got, err)
	}
	got, err = streamSSE[result](context.Background(), strings.NewReader("event: done\ndata: {\"n\":2}\n\nevent: done\ndata: \n\n"), nil)
	if err != nil || got == nil || got.N != 2 {
		t.Fatalf("empty done preserves prior value: %#v %v", got, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = streamSSE[result](ctx, strings.NewReader(input), nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("context precedence: %v", err)
	}
	_, err = streamSSE[result](ctx, &formalStreamsBadReader{}, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("context must precede read failure: %v", err)
	}
	_, err = streamSSE[result](context.Background(), io.LimitReader(strings.NewReader(""), 0), nil)
	if err == nil {
		t.Fatal("missing done accepted")
	}
}

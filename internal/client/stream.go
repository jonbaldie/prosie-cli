package client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// eventFrame collects one SSE event, including data spread over multiple lines.
type eventFrame struct {
	event string
	data  []string
}

func (f *eventFrame) accept(line string) bool {
	if line == "" {
		return true
	}
	if strings.HasPrefix(line, "event:") {
		f.event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
	} else if strings.HasPrefix(line, "data:") {
		f.data = append(f.data, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
	}
	return false
}

func (f *eventFrame) dispatch(consume func(string, string)) {
	consume(f.event, strings.Join(f.data, "\n"))
	f.event = ""
	f.data = nil
}

func readEvents(body io.Reader, consume func(string, string)) error {
	reader := bufio.NewReader(body)
	var frame eventFrame
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return err
		}
		if frame.accept(strings.TrimRight(line, "\r\n")) {
			frame.dispatch(consume)
		}
		if err == io.EOF {
			frame.dispatch(consume)
			return nil
		}
	}
}

type streamTokens struct {
	content strings.Builder
	onToken func(string)
}

func (t *streamTokens) accept(data string) {
	var payload struct {
		Delta string `json:"delta"`
	}
	if json.Unmarshal([]byte(data), &payload) != nil || payload.Delta == "" {
		return
	}
	t.content.WriteString(payload.Delta)
	if t.onToken != nil {
		t.onToken(payload.Delta)
	}
}

func streamError(data string) string {
	var payload struct {
		Message string `json:"message"`
	}
	if json.Unmarshal([]byte(data), &payload) == nil && payload.Message != "" {
		return payload.Message
	}
	return data
}

type streamEvents[T any] struct {
	tokens streamTokens
	done   *T
	err    error
}

func (s *streamEvents[T]) accept(event, data string) {
	data = strings.TrimSpace(data)
	switch event {
	case "delta":
		s.tokens.accept(data)
	case "done":
		var result T
		if json.Unmarshal([]byte(data), &result) == nil {
			s.done = &result
		}
	case "error":
		s.err = fmt.Errorf("stream error: %s", streamError(data))
	}
}

// streamSSE reads and decodes Server-Sent Events from an HTTP response stream.
func streamSSE[T any](ctx context.Context, body io.Reader, onToken func(string)) (*T, string, error) {
	state := streamEvents[T]{tokens: streamTokens{onToken: onToken}}
	err := readEvents(body, state.accept)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, state.tokens.content.String(), ctxErr
	}
	if err != nil {
		return nil, state.tokens.content.String(), err
	}
	if state.err != nil {
		return nil, state.tokens.content.String(), state.err
	}
	return state.done, state.tokens.content.String(), nil
}

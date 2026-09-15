package client

import (
	"bufio"
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

func readEvents(body io.Reader, flushEOF bool, consume func(string, string)) error {
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
			if flushEOF {
				frame.dispatch(consume)
			}
			return nil
		}
	}
}

type streamTokens struct {
	content strings.Builder
	onToken func(string)
}

func (t *streamTokens) accept(data string, allowEmpty bool) {
	var payload struct {
		Delta string `json:"delta"`
	}
	if json.Unmarshal([]byte(data), &payload) != nil {
		return
	}
	if payload.Delta == "" && !allowEmpty {
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

type generationEvents[T any] struct {
	tokens streamTokens
	done   *T
	err    error
}

func (s *generationEvents[T]) accept(event, data string) {
	switch event {
	case "delta":
		s.tokens.accept(data, false)
	case "done":
		var result T
		if json.Unmarshal([]byte(data), &result) == nil {
			s.done = &result
		}
	case "error":
		s.err = fmt.Errorf("stream error: %s", streamError(data))
	}
}

type chatEvents struct {
	tokens streamTokens
	done   *SendMessageResponse
	err    error
}

func (s *chatEvents) accept(event, data string) {
	data = strings.TrimSpace(data)
	switch event {
	case "delta":
		s.tokens.accept(data, true)
	case "done":
		var result SendMessageResponse
		if json.Unmarshal([]byte(data), &result) == nil {
			s.done = &result
		}
	case "error":
		s.err = chatStreamError(data)
	}
}

func chatStreamError(data string) error {
	var payload struct {
		Message string `json:"message"`
	}
	if json.Unmarshal([]byte(data), &payload) == nil && payload.Message != "" {
		return fmt.Errorf("%s", payload.Message)
	}
	return fmt.Errorf("stream error: %s", data)
}

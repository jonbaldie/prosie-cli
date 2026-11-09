package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestFormalChatRetainsCreatedConversation(t *testing.T) {
	for _, stream := range []bool{false, true} {
		t.Run(fmt.Sprint(stream), func(t *testing.T) {
			var paths []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				paths = append(paths, r.URL.Path)
				if r.URL.Path == "/api/stories/5/conversations" {
					fmt.Fprint(w, `{"id":77}`)
					return
				}
				http.Error(w, `{"message":"send failed"}`, http.StatusBadGateway)
			}))
			defer server.Close()
			var callback func(string)
			if stream {
				callback = func(string) {}
			}
			result, err := New(server.URL, "", server.Client()).Conversations().SendTurn(context.Background(), TurnTarget{BookID: "5"}, "hello", callback)
			suffix := ""
			if stream {
				suffix = "/stream"
			}
			if err == nil || !result.Created || result.ConversationID != "77" || !reflect.DeepEqual(paths, []string{"/api/stories/5/conversations", "/api/conversations/77/messages" + suffix}) {
				t.Fatalf("created turn not retained after send failure: result=%+v err=%v requests=%v", result, err, paths)
			}
		})
	}
	t.Run("whitespace skips creation but fails validation", func(t *testing.T) {
		calls := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++ }))
		defer server.Close()
		result, err := New(server.URL, "", server.Client()).Conversations().SendTurn(context.Background(), TurnTarget{ConversationID: "  "}, "hello", nil)
		if err == nil || result.Created || result.ConversationID != "  " || calls != 0 {
			t.Fatalf("result=%+v err=%v calls=%d", result, err, calls)
		}
	})
	t.Run("zero created id is sent", func(t *testing.T) {
		var paths []string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			paths = append(paths, r.URL.Path)
			if len(paths) == 1 {
				fmt.Fprint(w, `{"id":0}`)
				return
			}
			fmt.Fprint(w, `{"model":"m"}`)
		}))
		defer server.Close()
		result, err := New(server.URL, "", server.Client()).Conversations().SendTurn(context.Background(), TurnTarget{BookID: "5"}, "hello", nil)
		if err != nil || !result.Created || result.ConversationID != "0" || !reflect.DeepEqual(paths, []string{"/api/stories/5/conversations", "/api/conversations/0/messages"}) {
			t.Fatalf("result=%+v err=%v paths=%v", result, err, paths)
		}
	})
}

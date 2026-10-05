package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Conversation represents a chat thread associated with a book manuscript.
type Conversation struct {
	ID            int           `json:"id"`
	StoryID       int           `json:"story_id"`
	Title         *string       `json:"title"`
	Model         *string       `json:"model,omitempty"`
	UpToSceneID   *int          `json:"up_to_scene_id,omitempty"`
	Fidelity      string        `json:"fidelity"`
	ExpiresInDays *int          `json:"expires_in_days,omitempty"`
	Messages      []ChatMessage `json:"messages,omitempty"`
	CreatedAt     string        `json:"created_at,omitempty"`
	UpdatedAt     string        `json:"updated_at,omitempty"`
}

// DisplayTitle returns the conversation title or a friendly fallback.
func (c Conversation) DisplayTitle() string {
	if c.Title != nil && *c.Title != "" {
		return *c.Title
	}
	return fmt.Sprintf("Conversation %d", c.ID)
}

// ChatMessage represents a single turn in a novel chat conversation.
type ChatMessage struct {
	ID             int    `json:"id"`
	ConversationID int    `json:"conversation_id,omitempty"`
	Role           string `json:"role"`
	Content        string `json:"content"`
	CreatedAt      string `json:"created_at,omitempty"`
	UpdatedAt      string `json:"updated_at,omitempty"`
}

// ChatUsage holds token usage metrics for an LLM response.
type ChatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// SendMessageResponse represents the payload returned after sending a chat message.
type SendMessageResponse struct {
	Message *ChatMessage `json:"message"`
	Model   string       `json:"model"`
	Usage   ChatUsage    `json:"usage"`
}

// ListConversations fetches all chat conversations for a book story.
func (c *Conversations) ListConversations(ctx context.Context, bookID string) ([]Conversation, error) {
	bookID = strings.TrimSpace(bookID)
	if bookID == "" {
		return nil, errors.New("book ID is required")
	}

	path := fmt.Sprintf("/api/stories/%s/conversations", url.PathEscape(bookID))
	bodyBytes, err := c.transport.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var envelope struct {
		Data []Conversation `json:"data"`
	}
	var conversations []Conversation
	if err := json.Unmarshal(bodyBytes, &envelope); err == nil && envelope.Data != nil {
		conversations = envelope.Data
	} else if err := json.Unmarshal(bodyBytes, &conversations); err != nil {
		return nil, fmt.Errorf("failed to decode conversations response: %w", err)
	}

	return conversations, nil
}

// GetConversation fetches a single chat thread by ID, including its message turns.
func (c *Conversations) GetConversation(ctx context.Context, id string) (*Conversation, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("conversation ID is required")
	}

	path := fmt.Sprintf("/api/conversations/%s", url.PathEscape(id))
	bodyBytes, err := c.transport.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	conversation, err := decodeConversation(bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode conversation response: %w", err)
	}

	return &conversation, nil
}

// CreateConversation creates a new chat conversation thread under a book manuscript.
func (c *Conversations) CreateConversation(ctx context.Context, bookID string, title string) (*Conversation, error) {
	bookID = strings.TrimSpace(bookID)
	if bookID == "" {
		return nil, errors.New("book ID is required")
	}

	body := make(map[string]any)
	if title != "" {
		body["title"] = title
	}

	path := fmt.Sprintf("/api/stories/%s/conversations", url.PathEscape(bookID))
	bodyBytes, err := c.transport.request(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}

	conversation, err := decodeConversation(bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode create conversation response: %w", err)
	}

	return &conversation, nil
}

// DeleteConversation deletes a chat thread and its associated message turns.
func (c *Conversations) DeleteConversation(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("conversation ID is required")
	}

	path := fmt.Sprintf("/api/conversations/%s", url.PathEscape(id))
	return c.transport.execute(ctx, http.MethodDelete, path, nil)
}

// ExportConversation downloads the raw JSON conversation export.
func (c *Conversations) ExportConversation(ctx context.Context, id string) ([]byte, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("conversation ID is required")
	}

	path := fmt.Sprintf("/api/conversations/%s/export", url.PathEscape(id))
	return c.transport.request(ctx, http.MethodGet, path, nil)
}

// ImportConversation uploads a JSON conversation export stream to rebuild a chat thread in a book.
func (c *Conversations) ImportConversation(ctx context.Context, bookID string, file UploadFile) (*Conversation, error) {
	bookID = strings.TrimSpace(bookID)
	if bookID == "" {
		return nil, errors.New("book ID is required")
	}

	path := fmt.Sprintf("/api/stories/%s/conversations/import", url.PathEscape(bookID))
	req, err := file.withDefaultFilename("conversation.json").postRequest(ctx, c.transport, path, "file", nil)
	if err != nil {
		return nil, err
	}
	bodyBytes, err := c.transport.readResponse(req, "import conversation")
	if err != nil {
		return nil, err
	}

	conversation, err := decodeConversation(bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode import response: %w", err)
	}

	return &conversation, nil
}

// SendChatMessage sends a message turn to a conversation and receives the assistant response.
func (c *Conversations) SendChatMessage(ctx context.Context, conversationID string, message string) (*SendMessageResponse, error) {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return nil, errors.New("conversation ID is required")
	}

	body := map[string]string{
		"content": message,
	}

	path := fmt.Sprintf("/api/conversations/%s/messages", url.PathEscape(conversationID))
	bodyBytes, err := c.transport.request(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}

	var envelope struct {
		Data SendMessageResponse `json:"data"`
	}
	var sendResp SendMessageResponse
	if err := json.Unmarshal(bodyBytes, &envelope); err == nil && (envelope.Data.Message != nil || envelope.Data.Model != "") {
		sendResp = envelope.Data
	} else if err := json.Unmarshal(bodyBytes, &sendResp); err != nil {
		return nil, fmt.Errorf("failed to decode message response: %w", err)
	}

	return &sendResp, nil
}

// StreamChatMessage sends a message turn and receives tokens streamed over SSE.
func (c *Conversations) StreamChatMessage(ctx context.Context, conversationID string, message string, onToken func(string)) (*SendMessageResponse, error) {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return nil, errors.New("conversation ID is required")
	}

	body := map[string]string{
		"content": message,
	}

	path := fmt.Sprintf("/api/conversations/%s/messages/stream", url.PathEscape(conversationID))
	resp, err := c.transport.openSSEStream(ctx, path, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	result, accumulated, err := streamSSE[SendMessageResponse](ctx, resp.Body, onToken)
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = &SendMessageResponse{Message: &ChatMessage{
			Role: "assistant", Content: accumulated,
		}}
	}
	return result, nil
}

// TurnTarget selects where a message turn goes: an existing conversation, or a
// new conversation in a book when ConversationID is empty.
type TurnTarget struct {
	BookID         string
	ConversationID string
	Title          string // title for a new conversation
}

// TurnResult is the outcome of a message turn. ConversationID is set when the
// conversation exists, also when SendTurn returns an error.
type TurnResult struct {
	ConversationID string
	Created        bool
	Response       *SendMessageResponse
}

// SendTurn creates the conversation when necessary and sends one message turn.
// A nil onToken sends through the non-streaming endpoint.
func (c *Conversations) SendTurn(ctx context.Context, target TurnTarget, message string, onToken func(string)) (TurnResult, error) {
	result := TurnResult{ConversationID: target.ConversationID}
	if result.ConversationID == "" {
		conv, err := c.CreateConversation(ctx, target.BookID, target.Title)
		if err != nil {
			return result, err
		}
		result.ConversationID = strconv.Itoa(conv.ID)
		result.Created = true
	}

	var err error
	if onToken == nil {
		result.Response, err = c.SendChatMessage(ctx, result.ConversationID, message)
	} else {
		result.Response, err = c.StreamChatMessage(ctx, result.ConversationID, message, onToken)
	}
	return result, err
}

// Conversations owns conversations operations over the shared authenticated transport.
type Conversations struct{ transport *Client }

// Conversations returns the conversations module for this client.
func (c *Client) Conversations() *Conversations { return &Conversations{transport: c} }

func decodeConversation(data []byte) (Conversation, error) {
	var envelope struct {
		Data Conversation `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err == nil && envelope.Data.ID != 0 {
		return envelope.Data, nil
	}
	var conversation Conversation
	err := json.Unmarshal(data, &conversation)
	return conversation, err
}

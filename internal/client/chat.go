package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
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
func (c *Client) ListConversations(ctx context.Context, bookID string) ([]Conversation, error) {
	bookID = strings.TrimSpace(bookID)
	if bookID == "" {
		return nil, errors.New("book ID is required")
	}

	path := fmt.Sprintf("/api/stories/%s/conversations", bookID)
	req, err := c.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %w", path, err)
	}
	defer resp.Body.Close()

	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
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
func (c *Client) GetConversation(ctx context.Context, id string) (*Conversation, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("conversation ID is required")
	}

	path := fmt.Sprintf("/api/conversations/%s", id)
	req, err := c.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %w", path, err)
	}
	defer resp.Body.Close()

	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var envelope struct {
		Data Conversation `json:"data"`
	}
	var conversation Conversation
	if err := json.Unmarshal(bodyBytes, &envelope); err == nil && envelope.Data.ID != 0 {
		conversation = envelope.Data
	} else if err := json.Unmarshal(bodyBytes, &conversation); err != nil {
		return nil, fmt.Errorf("failed to decode conversation response: %w", err)
	}

	return &conversation, nil
}

// CreateConversation creates a new chat conversation thread under a book manuscript.
func (c *Client) CreateConversation(ctx context.Context, bookID string, title string) (*Conversation, error) {
	bookID = strings.TrimSpace(bookID)
	if bookID == "" {
		return nil, errors.New("book ID is required")
	}

	body := make(map[string]any)
	if title != "" {
		body["title"] = title
	}

	path := fmt.Sprintf("/api/stories/%s/conversations", bookID)
	req, err := c.NewRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %w", path, err)
	}
	defer resp.Body.Close()

	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var envelope struct {
		Data Conversation `json:"data"`
	}
	var conversation Conversation
	if err := json.Unmarshal(bodyBytes, &envelope); err == nil && envelope.Data.ID != 0 {
		conversation = envelope.Data
	} else if err := json.Unmarshal(bodyBytes, &conversation); err != nil {
		return nil, fmt.Errorf("failed to decode create conversation response: %w", err)
	}

	return &conversation, nil
}

// DeleteConversation deletes a chat thread and its associated message turns.
func (c *Client) DeleteConversation(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("conversation ID is required")
	}

	path := fmt.Sprintf("/api/conversations/%s", id)
	req, err := c.NewRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}

	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("request to %s failed: %w", path, err)
	}
	defer resp.Body.Close()

	return CheckResponse(resp)
}

// ExportConversation downloads the raw JSON conversation export.
func (c *Client) ExportConversation(ctx context.Context, id string) ([]byte, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("conversation ID is required")
	}

	path := fmt.Sprintf("/api/conversations/%s/export", id)
	req, err := c.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %w", path, err)
	}
	defer resp.Body.Close()

	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	return io.ReadAll(resp.Body)
}

// ImportConversation uploads a JSON conversation export to rebuild a chat thread in a book.
func (c *Client) ImportConversation(ctx context.Context, bookID string, filePath string) (*Conversation, error) {
	bookID = strings.TrimSpace(bookID)
	if bookID == "" {
		return nil, errors.New("book ID is required")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("failed to copy file data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	url := fmt.Sprintf("%s/api/stories/%s/conversations/import", c.BaseURL, bookID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("User-Agent", c.UserAgent)
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to import conversation failed: %w", err)
	}
	defer resp.Body.Close()

	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var envelope struct {
		Data Conversation `json:"data"`
	}
	var conversation Conversation
	if err := json.Unmarshal(bodyBytes, &envelope); err == nil && envelope.Data.ID != 0 {
		conversation = envelope.Data
	} else if err := json.Unmarshal(bodyBytes, &conversation); err != nil {
		return nil, fmt.Errorf("failed to decode import response: %w", err)
	}

	return &conversation, nil
}

// SendChatMessage sends a message turn to a conversation and receives the assistant response.
func (c *Client) SendChatMessage(ctx context.Context, conversationID string, message string) (*SendMessageResponse, error) {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return nil, errors.New("conversation ID is required")
	}

	body := map[string]string{
		"content": message,
	}

	path := fmt.Sprintf("/api/conversations/%s/messages", conversationID)
	req, err := c.NewRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %w", path, err)
	}
	defer resp.Body.Close()

	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
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
func (c *Client) StreamChatMessage(ctx context.Context, conversationID string, message string, onToken func(string)) (*SendMessageResponse, error) {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return nil, errors.New("conversation ID is required")
	}

	body := map[string]string{
		"content": message,
	}

	path := fmt.Sprintf("/api/conversations/%s/messages/stream", conversationID)
	req, err := c.NewRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %w", path, err)
	}
	defer resp.Body.Close()

	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	reader := bufio.NewReader(resp.Body)
	var currentEvent string
	var currentData strings.Builder
	var finalResp *SendMessageResponse
	var accumulatedContent strings.Builder
	var streamErr error

	dispatch := func() {
		if currentEvent == "" && currentData.Len() == 0 {
			return
		}
		dataStr := strings.TrimSpace(currentData.String())

		switch currentEvent {
		case "delta":
			var d struct {
				Delta string `json:"delta"`
			}
			if err := json.Unmarshal([]byte(dataStr), &d); err == nil {
				accumulatedContent.WriteString(d.Delta)
				if onToken != nil {
					onToken(d.Delta)
				}
			}
		case "done":
			var donePayload SendMessageResponse
			if err := json.Unmarshal([]byte(dataStr), &donePayload); err == nil {
				finalResp = &donePayload
			}
		case "error":
			var errPayload struct {
				Message string `json:"message"`
			}
			if err := json.Unmarshal([]byte(dataStr), &errPayload); err == nil && errPayload.Message != "" {
				streamErr = errors.New(errPayload.Message)
			} else {
				streamErr = fmt.Errorf("stream error: %s", dataStr)
			}
		}

		currentEvent = ""
		currentData.Reset()
	}

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("error reading stream: %w", err)
		}

		line = strings.TrimRight(line, "\r\n")

		if line == "" {
			dispatch()
		} else if strings.HasPrefix(line, "event:") {
			currentEvent = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			dataPart := strings.TrimPrefix(line, "data:")
			if strings.HasPrefix(dataPart, " ") {
				dataPart = dataPart[1:]
			}
			if currentData.Len() > 0 {
				currentData.WriteString("\n")
			}
			currentData.WriteString(dataPart)
		}

		if err == io.EOF {
			dispatch()
			break
		}
	}

	if streamErr != nil {
		return nil, streamErr
	}

	if finalResp == nil {
		finalResp = &SendMessageResponse{
			Message: &ChatMessage{
				Role:    "assistant",
				Content: accumulatedContent.String(),
			},
		}
	}

	return finalResp, nil
}

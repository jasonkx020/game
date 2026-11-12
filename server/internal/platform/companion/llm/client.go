package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Config struct {
	BaseURL    string
	APIKey     string
	Model      string
	TimeoutSec int
}

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type ToolDef struct {
	Type     string `json:"type"`
	Function struct {
		Name        string      `json:"name"`
		Description string      `json:"description"`
		Parameters  interface{} `json:"parameters"`
	} `json:"function"`
}

type chatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Tools    []ToolDef `json:"tools,omitempty"`
	Stream   bool      `json:"stream,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message      Message `json:"message"`
		Delta        Message `json:"delta"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
}

type Client struct {
	cfg    Config
	http   *http.Client
	enabled bool
}

func NewClient(cfg Config) *Client {
	timeout := time.Duration(cfg.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	base := strings.TrimRight(cfg.BaseURL, "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	enabled := cfg.APIKey != ""
	return &Client{
		cfg: Config{BaseURL: base, APIKey: cfg.APIKey, Model: cfg.Model, TimeoutSec: cfg.TimeoutSec},
		http: &http.Client{Timeout: timeout},
		enabled: enabled,
	}
}

func (c *Client) Enabled() bool { return c.enabled }

func (c *Client) Model() string {
	if c.cfg.Model != "" {
		return c.cfg.Model
	}
	return "gpt-4o-mini"
}

func (c *Client) Chat(ctx context.Context, messages []Message, tools []ToolDef) (Message, error) {
	if !c.enabled {
		return Message{Role: "assistant", Content: mockReply(messages)}, nil
	}
	body, _ := json.Marshal(chatRequest{Model: c.Model(), Messages: messages, Tools: tools})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Message{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	res, err := c.http.Do(req)
	if err != nil {
		return Message{}, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		b, _ := io.ReadAll(res.Body)
		return Message{}, fmt.Errorf("llm http %d: %s", res.StatusCode, string(b))
	}
	var out chatResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return Message{}, err
	}
	if len(out.Choices) == 0 {
		return Message{Role: "assistant", Content: ""}, nil
	}
	return out.Choices[0].Message, nil
}

func (c *Client) ChatStream(ctx context.Context, messages []Message, w io.Writer) error {
	if !c.enabled {
		_, err := fmt.Fprintf(w, "data: %s\n\n", ssePayload(mockReply(messages)))
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, "data: [DONE]\n\n")
		return err
	}
	body, _ := json.Marshal(chatRequest{Model: c.Model(), Messages: messages, Stream: true})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	req.Header.Set("Accept", "text/event-stream")
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		b, _ := io.ReadAll(res.Body)
		return fmt.Errorf("llm stream %d: %s", res.StatusCode, string(b))
	}
	scanner := bufio.NewScanner(res.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			_, _ = io.WriteString(w, "data: [DONE]\n\n")
			break
		}
		var chunk chatResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta.Content
		if delta == "" {
			continue
		}
		if _, err := fmt.Fprintf(w, "data: %s\n\n", ssePayload(delta)); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func ssePayload(text string) string {
	b, _ := json.Marshal(map[string]string{"content": text})
	return string(b)
}

func mockReply(messages []Message) string {
	last := ""
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			last = messages[i].Content
			break
		}
	}
	switch {
	case strings.Contains(last, "推荐"):
		return "根据你的偏好，我推荐试试打乌龟或六子冲！说「开一局打乌龟」我可以帮你开房～"
	case strings.Contains(last, "规则"):
		return "想了解哪个游戏的规则？比如「讲讲打乌龟规则」，我马上给你讲～"
	case strings.Contains(last, "开"):
		return "好的！告诉我玩哪个游戏、几个人，例如「开一局 4 人打乌龟」～"
	default:
		return "嗨！我是小龟，你的陪玩伴侣～想聊天、学规则还是开一局？随时跟我说！"
	}
}

package llm

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Candidate struct {
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

type Client struct {
	httpClient *http.Client
	apiKey     string
	model      string
	baseURL    string
}

func NewClient(apiKey, model, baseURL string) *Client {
	if model == "" {
		model = "deepseek-chat"
	}
	if baseURL == "" {
		baseURL = "https://api.deepseek.com/v1/chat/completions"
	}
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiKey:     apiKey,
		model:      model,
		baseURL:    baseURL,
	}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model          string        `json:"model"`
	Messages       []chatMessage `json:"messages"`
	Temperature    float64       `json:"temperature"`
	ResponseFormat *responseFmt  `json:"response_format,omitempty"`
}

type responseFmt struct {
	Type string `json:"type"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *Client) Recommend(industry string) ([]Candidate, error) {
	content, err := c.chat(systemPrompt, fmt.Sprintf("行业：%s", industry), true)
	if err != nil {
		return nil, err
	}

	var wrapper struct {
		Recommendations []Candidate `json:"recommendations"`
	}
	if err := json.Unmarshal([]byte(content), &wrapper); err != nil {
		var candidates []Candidate
		if err2 := json.Unmarshal([]byte(content), &candidates); err2 != nil {
			return nil, fmt.Errorf("llm json parse: %w (content: %s)", err, truncate(content, 200))
		}
		return candidates, nil
	}
	return wrapper.Recommendations, nil
}

func (c *Client) Chat(systemPrompt, userPrompt string) (string, error) {
	return c.chat(systemPrompt, userPrompt, false)
}

func (c *Client) chat(systemPrompt, userPrompt string, jsonMode bool) (string, error) {
	reqBody := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.3,
	}
	if jsonMode {
		reqBody.ResponseFormat = &responseFmt{Type: "json_object"}
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", c.baseURL, strings.NewReader(string(body)))
	if err != nil {
		return "", fmt.Errorf("llm request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("llm call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("llm read: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llm HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("llm parse: %w", err)
	}
	if chatResp.Error != nil {
		return "", fmt.Errorf("llm api error: %s", chatResp.Error.Message)
	}
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("llm: no choices returned")
	}
	return chatResp.Choices[0].Message.Content, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

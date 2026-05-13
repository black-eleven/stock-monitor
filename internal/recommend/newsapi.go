package recommend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Article struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	PublishedAt string `json:"publishedAt"`
	URL         string `json:"url"`
}

type newsAPIResponse struct {
	Status       string    `json:"status"`
	TotalResults int       `json:"totalResults"`
	Articles     []Article `json:"articles"`
}

type NewsAPIClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func NewNewsAPIClient(apiKey string) *NewsAPIClient {
	return &NewsAPIClient{
		apiKey:     apiKey,
		baseURL:    "https://newsapi.org/v2/everything",
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *NewsAPIClient) Search(query string, days int, pageSize int) ([]Article, error) {
	fromDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	u, _ := url.Parse(c.baseURL)
	u.RawQuery = url.Values{
		"q":        {query},
		"apiKey":   {c.apiKey},
		"sortBy":   {"popularity"},
		"pageSize": {fmt.Sprintf("%d", pageSize)},
		"from":     {fromDate},
		"language": {"en"},
	}.Encode()

	resp, err := c.httpClient.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("newsapi request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var apiErr struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		json.NewDecoder(resp.Body).Decode(&apiErr)
		return nil, fmt.Errorf("newsapi error %s: %s", apiErr.Code, apiErr.Message)
	}

	var result newsAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("newsapi decode: %w", err)
	}

	return result.Articles, nil
}

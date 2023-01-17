package models

type AIRequest struct {
	Model       string `json:"model"`
	Prompt      string `json:"prompt"`
	Temperature int64  `json:"temperature"`
	MaxTokens   int64  `json:"max_tokens"`
}

type AIResponse struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Text string `json:"text"`
}

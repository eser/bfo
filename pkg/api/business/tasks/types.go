package tasks

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Task struct {
	ID        string    `json:"id,omitempty"`
	Messages  []Message `json:"messages"`
	MaxTokens int       `json:"max_tokens,omitempty"`
}

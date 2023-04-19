package entity

type ChatConfig struct {
	Model            *Model
	Temperature      float32
	TopP             float32
	N                int
	Stop             []string
	MaxTokens        int
	PresencePenalty  float32
	FrequencyPenalty float32
}

type Chat struct {
	Id                   string
	UserId               string
	InitialSystemMessage *Message
	Messages             []*Message
	ErasedMessages       []*Message
	Status               string
	TokenUsage           int
	Config               *ChatConfig
}

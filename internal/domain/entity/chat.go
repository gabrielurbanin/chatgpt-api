package entity

import (
	"errors"

	"github.com/google/uuid"
)

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

func NewChat(userId string, initialSystemMessage *Message, chatConfig *ChatConfig) (*Chat, error) {
	chat := &Chat{
		Id:                   uuid.New().String(),
		UserId:               userId,
		InitialSystemMessage: initialSystemMessage,
		Status:               "active",
		Config:               chatConfig,
		TokenUsage:           0,
	}

	chat.AddMessage(initialSystemMessage)

	if err := chat.Validate(); err != nil {
		return nil, err
	}
	return chat, nil
}

func (c *Chat) Validate() error {
	if c.UserId == "" {
		return errors.New("User Id is empty")
	}

	if c.Status != "active" && c.Status != "ended" {
		return errors.New("Invalid status")
	}

	if c.Config.Temperature < 0 || c.Config.Temperature > 2 {
		return errors.New("Invalid temperature")
	}

	return nil
}

func (c *Chat) AddMessage(m *Message) error {
	if c.Status == "ended" {
		return errors.New("Chat is already full. No more messages allowed")
	}

	for {
		if c.Config.Model.GetMaxTokens() >= c.TokenUsage+m.GetTokensUsed() {
			c.Messages = append(c.Messages, m)
			c.RefreshTokenUsage()
			break
		}

		c.ErasedMessages = append(c.ErasedMessages, c.Messages[0])
		c.Messages = c.Messages[1:]
		c.RefreshTokenUsage()
	}

	return nil
}

func (c *Chat) GetMessages() []*Message {
	return c.Messages
}

func (c *Chat) CountMessages() int {
	return len(c.Messages)
}

func (c *Chat) End() {
	c.Status = "ended"
}

func (c *Chat) RefreshTokenUsage() {
	c.TokenUsage = 0

	for m := range c.Messages {
		c.TokenUsage += c.Messages[m].GetTokensUsed()
	}
}

package chatcompletionstream

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/gabrielurbanin/chatgpt-api/internal/domain/entity"
	"github.com/gabrielurbanin/chatgpt-api/internal/domain/gateway"
	openai "github.com/sashabaranov/go-openai"
)

type ChatCompletionUseCase struct {
	ChatGateway  gateway.ChatGateway
	OpenAiClient *openai.Client
	Stream       chan ChatCompletionOutputDTO
}

type ChatCompletionConfigInputDTO struct {
	Model                string
	ModelMaxTokens       int
	Temperature          float32
	TopP                 float32
	N                    int
	Stop                 []string
	MaxTokens            int
	PresencePenalty      float32
	FrequencyPenalty     float32
	InitialSystemMessage string
}

type ChatCompletionInputDTO struct {
	ChatId      string
	UserId      string
	UserMessage string
	Config      ChatCompletionConfigInputDTO
}

type ChatCompletionOutputDTO struct {
	ChatId  string
	UserId  string
	Content string
}

func NewChatCompletionUseCase(chatGateway gateway.ChatGateway, openAiClient *openai.Client, stream chan ChatCompletionOutputDTO) *ChatCompletionUseCase {
	return &ChatCompletionUseCase{
		ChatGateway:  chatGateway,
		OpenAiClient: openAiClient,
		Stream:       stream,
	}
}

func (useCase *ChatCompletionUseCase) Execute(ctx context.Context, input ChatCompletionInputDTO) (*ChatCompletionOutputDTO, error) {
	chat, err := useCase.ChatGateway.FindChatById(ctx, input.ChatId)

	if err != nil {
		if err.Error() == "Chat not found" {
			chat, err = createNewChat(input)

			if err != nil {
				return nil, errors.New("Could not create a new chat: " + err.Error())
			}

			err = useCase.ChatGateway.CreateChat(ctx, chat)

			if err != nil {
				return nil, errors.New("Could not persist new chat: " + err.Error())
			}
		} else {
			return nil, errors.New("Could not fetch existing chat: " + err.Error())
		}
	}

	userMessage, err := entity.NewMessage("user", input.UserMessage, chat.Config.Model)
	if err != nil {
		return nil, errors.New("Could not create user message: " + err.Error())
	}

	err = chat.AddMessage(userMessage)
	if err != nil {
		return nil, errors.New("Could not add user message to chat: " + err.Error())
	}

	messages := []openai.ChatCompletionMessage{}
	for _, msg := range chat.Messages {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	request := openai.ChatCompletionRequest{
		Model:            chat.Config.Model.Name,
		Messages:         messages,
		MaxTokens:        chat.Config.MaxTokens,
		Temperature:      chat.Config.Temperature,
		TopP:             chat.Config.TopP,
		PresencePenalty:  chat.Config.PresencePenalty,
		FrequencyPenalty: chat.Config.FrequencyPenalty,
		Stop:             chat.Config.Stop,
		Stream:           true,
	}

	resp, err := useCase.OpenAiClient.CreateChatCompletionStream(ctx, request)
	if err != nil {
		return nil, errors.New("Failed to send request to openai client: " + err.Error())
	}

	var fullResponse strings.Builder
	for {
		response, err := resp.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, errors.New("Error streaming response: " + err.Error())
		}

		fullResponse.WriteString(response.Choices[0].Delta.Content)
		useCase.Stream <- ChatCompletionOutputDTO{
			ChatId:  input.ChatId,
			UserId:  input.UserId,
			Content: fullResponse.String(),
		}
	}

	assistant, err := entity.NewMessage("assistant", fullResponse.String(), chat.Config.Model)
	if err != nil {
		return nil, errors.New("Error creating assistant chat: " + err.Error())
	}

	err = chat.AddMessage(assistant)
	if err != nil {
		return nil, errors.New("Could not add assistant response to chat: " + err.Error())
	}

	err = useCase.ChatGateway.SaveChat(ctx, chat)
	if err != nil {
		return nil, errors.New("Could not save chat: " + err.Error())
	}

	return &ChatCompletionOutputDTO{
		ChatId:  input.ChatId,
		UserId:  input.UserId,
		Content: fullResponse.String(),
	}, nil
}

func createNewChat(input ChatCompletionInputDTO) (*entity.Chat, error) {
	model := entity.NewModel(input.Config.Model, input.Config.MaxTokens)

	chatConfig := &entity.ChatConfig{
		Temperature:      input.Config.Temperature,
		TopP:             input.Config.TopP,
		N:                input.Config.N,
		Stop:             input.Config.Stop,
		MaxTokens:        input.Config.MaxTokens,
		PresencePenalty:  input.Config.PresencePenalty,
		FrequencyPenalty: input.Config.FrequencyPenalty,
		Model:            model,
	}

	initialMessage, err := entity.NewMessage("system", input.Config.InitialSystemMessage, model)
	if err != nil {
		return nil, errors.New("Could not create initial message:" + err.Error())
	}

	chat, err := entity.NewChat(input.UserId, initialMessage, chatConfig)
	if err != nil {
		return nil, errors.New("Could not create new chat:" + err.Error())
	}

	return chat, nil
}

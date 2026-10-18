package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

func main() {
	ctx := context.Background()
	offset := 0

	telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN")

	if telegramToken == "" {
		fmt.Println("Укажите переменную TELEGRAM_BOT_TOKEN")
		return
	}

	openAIAPIKey := os.Getenv("OPENAI_API_KEY")
	if openAIAPIKey == "" {
		fmt.Println("Укажите переменную OPENAI_API_KEY")
		return
	}
	strapiBaseURL := os.Getenv("STRAPI_BASE_URL")
	if strapiBaseURL == "" {
		fmt.Println("Укажите переменную STRAPI_BASE_URL")
		return
	}
	strapiAPIToken := os.Getenv("STRAPI_API_TOKEN")
	if strapiAPIToken == "" {
		fmt.Println("Укажите переменную STRAPI_API_TOKEN")
		return
	}

	telegramClient := TelegramClient{
		token: telegramToken,
	}
	threadID := 0
	if value := os.Getenv("TELEGRAM_LOG_THREAD_ID"); value != "" {
		var err error
		threadID, err = strconv.Atoi(value)
		if err != nil || threadID <= 0 {
			fmt.Println("TELEGRAM_LOG_THREAD_ID должен быть положительным числом")
			return
		}
	}
	notifier := &PublicationNotifier{client: &telegramClient, chatID: os.Getenv("TELEGRAM_LOG_CHAT_ID"), siteURL: os.Getenv("SITE_BASE_URL"), threadID: threadID}
	openAIClient := openai.NewClient(option.WithAPIKey(openAIAPIKey))
	productExtractor := OpenAIProductExtractor{client: &openAIClient}
	strapiClient, err := NewStrapiClient(
		strapiBaseURL,
		strapiAPIToken,
		&http.Client{Timeout: 2 * time.Minute},
	)
	if err != nil {
		fmt.Println("Strapi client error:", err)
		return
	}

	downloadDir := "downloads"

	errCreateDir := os.MkdirAll(downloadDir, 0750)
	if errCreateDir != nil {
		fmt.Printf("error create dir: %s", errCreateDir)
		return
	}

	for {
		telegramResponse, err := telegramClient.getUpdates(offset)

		if err != nil {
			fmt.Println("Telegram request error:", err)
			return
		}

		for _, update := range telegramResponse.Result {
			message := update.Message
			if strings.SplitN(message.Text, "@", 2)[0] == "/chatid" && message.Chat.ID != 0 {
				if err := telegramClient.sendMessage(ctx, strconv.FormatInt(message.Chat.ID, 10), chatIDText(message), message.MessageThreadID); err != nil {
					fmt.Println("Chat ID reply failed")
				}
			}
		}
		nextOffset, processError := processUpdateBatch(
			telegramResponse.Result,
			offset,
			func(post TelegramPost) error {
				return processPost(
					ctx,
					&telegramClient,
					productExtractor,
					strapiClient,
					post,
					downloadDir,
					notifier,
				)
			},
		)
		if processError != nil {
			fmt.Println("Process post error:", processError)
		}
		offset = nextOffset

	}
}

func processUpdateBatch(
	updates []Update,
	currentOffset int,
	process func(TelegramPost) error,
) (int, error) {
	candidateOffset := currentOffset
	for _, update := range updates {
		if update.UpdateID+1 > candidateOffset {
			candidateOffset = update.UpdateID + 1
		}
	}

	var failures []error
	for _, post := range groupUpdates(updates) {
		if err := process(post); err != nil {
			failures = append(failures, fmt.Errorf("process post %v: %w", post.UpdateIDs, err))
		}
	}

	return candidateOffset, errors.Join(failures...)
}

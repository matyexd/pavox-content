package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
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
				)
			},
		)
		if processError != nil {
			fmt.Println("Process post error:", processError)
		} else {
			offset = nextOffset
		}

		if len(telegramResponse.Result) == 0 {
			fmt.Println("No new posts")
		}
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

	for _, post := range groupUpdates(updates) {
		if err := process(post); err != nil {
			return currentOffset, fmt.Errorf("process post: %w", err)
		}
	}

	return candidateOffset, nil
}

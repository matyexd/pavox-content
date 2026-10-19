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

	allowedChannels, err := parseAllowedChannelIDs(os.Getenv("TELEGRAM_ALLOWED_CHANNEL_IDS"))
	if err != nil {
		fmt.Println(err)
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
			allowedChannels,
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
	allowedChannels map[int64]bool,
	process func(TelegramPost) error,
) (int, error) {
	candidateOffset := currentOffset
	for _, update := range updates {
		if update.UpdateID+1 > candidateOffset {
			candidateOffset = update.UpdateID + 1
		}
	}

	// Filter before grouping so rejected album parts cannot enter an allowed post.
	allowedUpdates := make([]Update, 0, len(updates))
	for _, update := range updates {
		id := update.ChannelPost.Chat.ID
		if id < 0 && allowedChannels[id] {
			allowedUpdates = append(allowedUpdates, update)
		}
	}

	var failures []error
	for _, post := range groupUpdates(allowedUpdates) {
		if err := process(post); err != nil {
			failures = append(failures, fmt.Errorf("process post %v: %w", post.UpdateIDs, err))
		}
	}

	return candidateOffset, errors.Join(failures...)
}

// An absent or malformed allowlist must never enable unrestricted publication.
func parseAllowedChannelIDs(value string) (map[int64]bool, error) {
	allowed := make(map[int64]bool)
	for _, part := range strings.Split(value, ",") {
		id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil || id >= 0 {
			return nil, fmt.Errorf("TELEGRAM_ALLOWED_CHANNEL_IDS должен содержать ID каналов (отрицательные числа), разделённые запятыми")
		}
		allowed[id] = true
	}
	return allowed, nil
}

package main

import (
	"fmt"
	"os"
)

func main() {
	offset := 0

	token := os.Getenv("TELEGRAM_BOT_TOKEN")

	if token == "" {
		fmt.Println("Укажите переменную token")
		return
	}

	client := TelegramClient{
		token: token,
	}

	downloadDir := "downloads"

	errCreateDir := os.MkdirAll(downloadDir, 0750)
	if errCreateDir != nil {
		fmt.Printf("error create dir: %s", errCreateDir)
		return
	}

	for {
		telegramResponse, err := client.getUpdates(offset)

		if err != nil {
			fmt.Println("Telegram request error:", err)
			return
		}

		nextOffset, processError := processUpdateBatch(
			telegramResponse.Result,
			offset,
			func(post TelegramPost) error {
				return processPost(&client, post, downloadDir)
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

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

	for {

		handleError := false
		telegramResponse, err := client.getUpdates(offset)

		if err != nil {
			fmt.Println("Telegram request error:", err)
			return
		}

		telegramPosts := groupUpdates(telegramResponse.Result)
		candidateOffset := offset

		for _, post := range telegramResponse.Result {
			if post.UpdateID+1 > candidateOffset {
				candidateOffset = post.UpdateID + 1
			}
		}

		for _, post := range telegramPosts {
			errorProcess := processPost(&client, post)
			if errorProcess != nil {
				fmt.Println("Process post error:", errorProcess)
				handleError = true
				break
			}
		}

		if !handleError {
			offset = candidateOffset
		}

		if len(telegramResponse.Result) == 0 {
			fmt.Println("No new posts")
		}
	}
}

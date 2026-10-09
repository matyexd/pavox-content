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

		telegramResponse, err := client.getUpdates(offset)

		if err != nil {
			fmt.Println("Telegram request error:", err)
			return
		}

		for _, post := range telegramResponse.Result {
			errorProcess := processUpdate(&client, post)
			if errorProcess != nil {
				fmt.Println("Process update error:", errorProcess)
				break
			}
			offset = post.UpdateID + 1
		}

		telegramPosts := processPost(telegramResponse.Result)
		fmt.Println(telegramPosts)

		if len(telegramResponse.Result) == 0 {
			fmt.Println("No new posts")
		}
	}
}

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
			fmt.Printf("Caption: %s \n", post.ChannelPost.Caption)
			fmt.Printf("Count variant photo: %d \n", len(post.ChannelPost.Photo))
			for index, photo := range post.ChannelPost.Photo {
				fmt.Printf("#%d \n", index+1)
				fmt.Printf("Width: %d \n", photo.Width)
				fmt.Printf("Height: %d \n", photo.Height)
			}

			if len(post.ChannelPost.Photo) > 0 {
				file, errorGetFile := client.getFile(post.ChannelPost.Photo[len(post.ChannelPost.Photo)-1].FileID)
				if errorGetFile != nil {
					fmt.Println("Error getting file:", errorGetFile)
					break
				}

				fileName := file.Result.FileUniqueID + ".jpg"

				newFile, errorCreate := os.Create(fileName)
				if errorCreate != nil {
					fmt.Println("Error create file:", errorCreate)
					break
				}

				downloadError := client.downloadTelegramFile(file.Result, newFile)
				closeError := newFile.Close()

				if downloadError != nil {
					fmt.Println("Error downloading file:", downloadError)
					break
				}

				if closeError != nil {
					fmt.Println("Error closing file:", closeError)
					break
				}
			}
			fmt.Println()
			offset = post.UpdateID + 1
		}

		if len(telegramResponse.Result) == 0 {
			fmt.Println("No new posts")
		}
	}
}

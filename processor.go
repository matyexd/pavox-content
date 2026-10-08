package main

import (
	"fmt"
	"os"
)

func processUpdate(client *TelegramClient, post Update) error {
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
			return fmt.Errorf("get file: %w", errorGetFile)
		}

		fileName := file.Result.FileUniqueID + ".jpg"

		newFile, errorCreate := os.Create(fileName)
		if errorCreate != nil {
			return fmt.Errorf("create file: %w", errorCreate)
		}

		downloadError := client.downloadTelegramFile(file.Result, newFile)
		closeError := newFile.Close()

		if downloadError != nil {
			return fmt.Errorf("download file: %w", downloadError)
		}

		if closeError != nil {
			return fmt.Errorf("close file: %w", closeError)
		}
	}

	return nil
}

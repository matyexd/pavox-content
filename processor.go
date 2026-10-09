package main

import (
	"fmt"
	"os"
	"strconv"
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

func processPost(updates []Update) []TelegramPost {
	posts := make(map[string][]Update)
	order := make([]string, 0)

	telegramPosts := make([]TelegramPost, 0)

	for _, post := range updates {
		var key string
		if post.ChannelPost.MediaGroupID == "" {
			key = "single:" + strconv.Itoa(post.UpdateID)
		} else {
			key = post.ChannelPost.MediaGroupID
		}

		if _, exists := posts[key]; !exists {
			order = append(order, key)
		}

		posts[key] = append(posts[key], post)
	}

	for _, key := range order {
		var telegramPost TelegramPost
		groupPosts := posts[key]
		telegramPost.MediaGroupID = groupPosts[0].ChannelPost.MediaGroupID

		for _, post := range groupPosts {
			if len(post.ChannelPost.Photo) > 0 {
				telegramPost.Photos = append(telegramPost.Photos, post.ChannelPost.Photo[len(post.ChannelPost.Photo)-1])
			}
			telegramPost.UpdateIDs = append(telegramPost.UpdateIDs, post.UpdateID)
		}

		for _, group := range groupPosts {
			if group.ChannelPost.Caption != "" {
				telegramPost.Caption = group.ChannelPost.Caption
				break
			}
		}

		for _, group := range groupPosts {
			if group.ChannelPost.Text != "" {
				telegramPost.Text = group.ChannelPost.Text
				break
			}
		}

		telegramPosts = append(telegramPosts, telegramPost)
	}

	return telegramPosts
}

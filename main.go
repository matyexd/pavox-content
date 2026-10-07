package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const TOKEN string = "8611116635:AAH9EnppPONEAZvGTQTqPz1JY509CJu7GSM"

type ChannelPost struct {
	MessageId int    `json:"message_id"`
	Text      string `json:"text"`
}

type Update struct {
	UpdateID    int         `json:"update_id"`
	ChannelPost ChannelPost `json:"channel_post"`
}

type TelegramResponse struct {
	Ok     bool     `json:"ok"`
	Result []Update `json:"result"`
}

func main() {
	offset := 0

	for {
		start := time.Now()
		url := "https://api.telegram.org/bot" + TOKEN + "/getUpdates" + "?offset=" + strconv.Itoa(offset) + "&timeout=30"
		response, err := http.Get(url)
		fmt.Println("Request took:", time.Since(start))

		if err != nil {
			fmt.Println("Request error:", err)
			return
		}

		body, err := io.ReadAll(response.Body)

		response.Body.Close()

		if err != nil {
			fmt.Println("Read error:", err)
			return
		}

		var telegramResponse TelegramResponse
		err = json.Unmarshal(body, &telegramResponse)

		if err != nil {
			fmt.Println("JSON error:", err)
			return
		}

		for _, post := range telegramResponse.Result {
			fmt.Printf("Update ID: %d \n", post.UpdateID)
			fmt.Printf("Text: %s \n", post.ChannelPost.Text)
			fmt.Println()
			offset = post.UpdateID + 1
		}

		if len(telegramResponse.Result) == 0 {
			fmt.Println("No new posts")
		}
	}
}

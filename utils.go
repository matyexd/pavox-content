package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func telegramRequest[T any](url string) (TelegramResponse[T], error) {
	var telegramResponse TelegramResponse[T]

	start := time.Now()
	response, err := http.Get(url)
	fmt.Println("Request took:", time.Since(start))

	if err != nil {
		return telegramResponse, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)

	if err != nil {
		return telegramResponse, err
	}

	err = json.Unmarshal(body, &telegramResponse)

	if err != nil {
		return telegramResponse, err
	}

	if !telegramResponse.OK {
		return telegramResponse, fmt.Errorf(
			"telegram API error %d: %s",
			telegramResponse.ErrorCode,
			telegramResponse.Description,
		)
	}

	return telegramResponse, err
}

func telegramRequestBinaryFile(url string, dst io.Writer) error {
	start := time.Now()
	response, err := http.Get(url)

	if err != nil {
		return err
	}
	defer response.Body.Close()
	fmt.Println("Request took:", time.Since(start))

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"telegram API error: %s",
			response.Status,
		)
	}

	_, err = io.Copy(dst, response.Body)

	return err
}

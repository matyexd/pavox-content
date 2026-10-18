package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func telegramRequest[T any](url string) (TelegramResponse[T], error) {
	var telegramResponse TelegramResponse[T]

	response, err := http.Get(url)

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
	response, err := http.Get(url)

	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"telegram API error: %s",
			response.Status,
		)
	}

	_, err = io.Copy(dst, response.Body)

	return err
}

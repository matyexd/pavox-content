package main

import (
	"io"
	"net/url"
	"strconv"
)

const telegramAPIBaseURL string = "https://api.telegram.org"

func (client *TelegramClient) getUpdates(offset int) (TelegramResponse[[]Update], error) {
	requestURL := telegramAPIBaseURL + "/bot" + client.token + "/getUpdates" + "?offset=" + strconv.Itoa(offset) + "&timeout=30&allowed_updates=" + url.QueryEscape(`["message","channel_post"]`)

	return telegramRequest[[]Update](requestURL)
}

func (client *TelegramClient) getFile(fileID string) (TelegramResponse[TelegramFile], error) {
	url := telegramAPIBaseURL + "/bot" + client.token + "/getFile" + "?file_id=" + fileID

	return telegramRequest[TelegramFile](url)
}

func (client *TelegramClient) downloadTelegramFile(file TelegramFile, newFile io.Writer) error {
	downloadURL := telegramAPIBaseURL + "/file/bot" + client.token + "/" + file.FilePath

	return telegramRequestBinaryFile(downloadURL, newFile)
}

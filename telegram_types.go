package main

type TelegramClient struct {
	token string
}

type PhotoSize struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	FileSize     int    `json:"file_size"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
}

type ChannelPost struct {
	MessageID    int         `json:"message_id"`
	Text         string      `json:"text"`
	Caption      string      `json:"caption"`
	Photo        []PhotoSize `json:"photo"`
	MediaGroupID string      `json:"media_group_id"`
}

type Update struct {
	UpdateID    int         `json:"update_id"`
	ChannelPost ChannelPost `json:"channel_post"`
}

type TelegramFile struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	FileSize     int64  `json:"file_size"`
	FilePath     string `json:"file_path"`
}

type TelegramResponse[T any] struct {
	OK          bool   `json:"ok"`
	Result      T      `json:"result"`
	Description string `json:"description"`
	ErrorCode   int    `json:"error_code"`
}

type TelegramPost struct {
	Caption      string      `json:"caption"`
	MediaGroupID string      `json:"media_group_id"`
	Photos       []PhotoSize `json:"photos"`
	UpdateIDs    []int       `json:"update_ids"`
	Text         string      `json:"text"`
}

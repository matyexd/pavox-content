package main

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestGroupUpdatesIgnoresChatCommands(t *testing.T) {
	updates := []Update{
		{UpdateID: 1, Message: ChannelPost{MessageID: 5, Text: "/chatid", Chat: TelegramChat{ID: -100123}}},
		{UpdateID: 2, ChannelPost: ChannelPost{MessageID: 6, Text: "Товар"}},
	}
	posts := groupUpdates(updates)
	if len(posts) != 1 || posts[0].Text != "Товар" {
		t.Fatalf("unexpected posts: %+v", posts)
	}
	next, err := processUpdateBatch(updates[:1], 0, func(TelegramPost) error {
		t.Fatal("chat command must not be imported")
		return nil
	})
	if err != nil || next != 2 {
		t.Fatalf("offset=%d, error=%v", next, err)
	}
}

func TestNotificationErrorRedactsCredentials(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "123:secret")
	t.Setenv("STRAPI_API_TOKEN", "strapi-secret")
	got := notificationError(fmt.Errorf("request https://api.telegram.org/bot123:secret/getFile: strapi-secret"))
	if strings.Contains(got, "123:secret") || strings.Contains(got, "strapi-secret") {
		t.Fatalf("credentials leaked: %s", got)
	}
	if !strings.Contains(got, "getFile") {
		t.Fatal("diagnostic context lost")
	}
}

func TestNotificationErrorTruncatesLongErrors(t *testing.T) {
	got := notificationError(errors.New(strings.Repeat("я", 5000)))
	if len([]rune(got)) != 1201 {
		t.Fatalf("unexpected diagnostic length: %d", len([]rune(got)))
	}
}

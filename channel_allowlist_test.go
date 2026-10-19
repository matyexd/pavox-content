package main

import (
	"context"
	"os"
	"testing"
)

func TestParseAllowedChannelIDs(t *testing.T) {
	for _, value := range []string{"", " ", "0", "123", "-1001,", "-1001,abc", "-9223372036854775809"} {
		if _, err := parseAllowedChannelIDs(value); err == nil {
			t.Errorf("accepted invalid allowlist %q", value)
		}
	}
	allowed, err := parseAllowedChannelIDs(" -1001, -1002,-1001 ")
	if err != nil || len(allowed) != 2 || !allowed[-1001] || !allowed[-1002] {
		t.Fatalf("unexpected allowlist %v: %v", allowed, err)
	}
}

func TestRejectedChannelsNeverReachProcessing(t *testing.T) {
	updates := []Update{
		{UpdateID: 10, ChannelPost: ChannelPost{Chat: TelegramChat{ID: -999}, Text: "product", Photo: []PhotoSize{{FileID: "foreign-photo"}}}},
		{UpdateID: 11, ChannelPost: ChannelPost{Text: "missing source"}},
		{UpdateID: 12, Message: ChannelPost{Chat: TelegramChat{ID: -1001}, Text: "private message"}},
	}
	for _, allowed := range []map[int64]bool{nil, {-1001: true}} {
		dir := t.TempDir()
		next, err := processUpdateBatch(updates, 0, allowed, func(post TelegramPost) error {
			t.Error("rejected post reached processing")
			// Nil clients would panic if any file, LLM, or Strapi method were called.
			return processPost(context.Background(), nil, nil, nil, post, dir)
		})
		if err != nil || next != 13 {
			t.Fatalf("offset=%d error=%v", next, err)
		}
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) != 0 {
			t.Fatalf("unexpected files: %v, %v", entries, err)
		}
	}
}

func TestChannelFilteringAndAlbumIsolation(t *testing.T) {
	updates := []Update{
		{UpdateID: 1, ChannelPost: ChannelPost{Chat: TelegramChat{ID: -1001}, MediaGroupID: "same", Caption: "first"}},
		{UpdateID: 2, ChannelPost: ChannelPost{Chat: TelegramChat{ID: -999}, MediaGroupID: "same", Photo: []PhotoSize{{FileID: "foreign"}}}},
		{UpdateID: 3, ChannelPost: ChannelPost{Chat: TelegramChat{ID: -1002}, MediaGroupID: "same", Caption: "second"}},
		{UpdateID: 4, ChannelPost: ChannelPost{Chat: TelegramChat{ID: -1001}, MediaGroupID: "same", Photo: []PhotoSize{{FileID: "allowed"}}}},
	}
	var posts []TelegramPost
	next, err := processUpdateBatch(updates, 0, map[int64]bool{-1001: true, -1002: true}, func(post TelegramPost) error {
		posts = append(posts, post)
		return nil
	})
	if err != nil || next != 5 || len(posts) != 2 {
		t.Fatalf("posts=%+v offset=%d error=%v", posts, next, err)
	}
	if posts[0].ChatID != -1001 || posts[0].Caption != "first" || len(posts[0].Photos) != 1 || posts[0].Photos[0].FileID != "allowed" || len(posts[0].UpdateIDs) != 2 {
		t.Fatalf("unexpected first album: %+v", posts[0])
	}
	if posts[1].ChatID != -1002 || posts[1].Caption != "second" || len(posts[1].Photos) != 0 {
		t.Fatalf("unexpected second album: %+v", posts[1])
	}
}

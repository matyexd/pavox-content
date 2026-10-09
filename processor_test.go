package main

import (
	"slices"
	"testing"
)

func TestProcessPostSingleTextUpdate(t *testing.T) {
	updates := []Update{
		{
			UpdateID: 100,
			ChannelPost: ChannelPost{
				Text: "Новый текстовый пост",
			},
		},
	}

	posts := groupUpdates(updates)

	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}

	post := posts[0]

	if post.Text != "Новый текстовый пост" {
		t.Errorf(
			"expected text %q, got %q",
			"Новый текстовый пост",
			post.Text,
		)
	}

	if len(post.UpdateIDs) != 1 || post.UpdateIDs[0] != 100 {
		t.Errorf("expected update ID 100, got %v", post.UpdateIDs)
	}

	if post.MediaGroupID != "" {
		t.Errorf(
			"expected empty media group ID, got %q",
			post.MediaGroupID,
		)
	}
}

func TestProcessPostGroupsAlbum(t *testing.T) {
	updates := []Update{
		{
			UpdateID: 200,
			ChannelPost: ChannelPost{
				MediaGroupID: "album-1",
				Caption:      "",
				Photo: []PhotoSize{
					{
						FileID: "small-1",
						Width:  100,
						Height: 100,
					},
					{
						FileID: "large-1",
						Width:  1000,
						Height: 1000,
					},
				},
			},
		},
		{
			UpdateID: 201,
			ChannelPost: ChannelPost{
				MediaGroupID: "album-1",
				Caption:      "Описание альбома",
				Photo: []PhotoSize{
					{
						FileID: "small-2",
						Width:  100,
						Height: 100,
					},
					{
						FileID: "large-2",
						Width:  1000,
						Height: 1000,
					},
				},
			},
		},
		{
			UpdateID: 202,
			ChannelPost: ChannelPost{
				MediaGroupID: "album-1",
				Caption:      "",
				Photo: []PhotoSize{
					{
						FileID: "small-3",
						Width:  100,
						Height: 100,
					},
					{
						FileID: "large-3",
						Width:  1000,
						Height: 1000,
					},
				},
			},
		},
	}

	posts := groupUpdates(updates)

	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}

	post := posts[0]

	if post.MediaGroupID != "album-1" {
		t.Errorf("expected media group ID %q, got %q", "album-1", post.MediaGroupID)
	}

	if post.Caption != "Описание альбома" {
		t.Errorf(
			"expected caption %q, got %q",
			"Описание альбома",
			post.Caption,
		)
	}

	expectedUpdateIDs := []int{200, 201, 202}
	if !slices.Equal(post.UpdateIDs, expectedUpdateIDs) {
		t.Errorf("expected update IDs %v, got %v", expectedUpdateIDs, post.UpdateIDs)
	}

	if len(post.Photos) != 3 {
		t.Fatalf("expected 3 photos, got %d", len(post.Photos))
	}

	expectedFileIDs := []string{"large-1", "large-2", "large-3"}
	for index, expectedFileID := range expectedFileIDs {
		if post.Photos[index].FileID != expectedFileID {
			t.Errorf(
				"expected photo %d file ID %q, got %q",
				index,
				expectedFileID,
				post.Photos[index].FileID,
			)
		}
	}
}

func TestProcessPostPreservesMixedPostOrder(t *testing.T) {
	updates := []Update{
		{
			UpdateID: 300,
			ChannelPost: ChannelPost{
				Text: "Первый одиночный пост",
			},
		},
		{
			UpdateID: 301,
			ChannelPost: ChannelPost{
				MediaGroupID: "album-2",
				Caption:      "Описание второго альбома",
				Photo: []PhotoSize{
					{FileID: "small-a", Width: 100, Height: 100},
					{FileID: "large-a", Width: 1000, Height: 1000},
				},
			},
		},
		{
			UpdateID: 302,
			ChannelPost: ChannelPost{
				MediaGroupID: "album-2",
				Photo: []PhotoSize{
					{FileID: "small-b", Width: 100, Height: 100},
					{FileID: "large-b", Width: 1000, Height: 1000},
				},
			},
		},
		{
			UpdateID: 303,
			ChannelPost: ChannelPost{
				Text: "Последний одиночный пост",
			},
		},
	}

	posts := groupUpdates(updates)

	if len(posts) != 3 {
		t.Fatalf("expected 3 posts, got %d", len(posts))
	}

	if posts[0].Text != "Первый одиночный пост" {
		t.Errorf("expected first single post first, got text %q", posts[0].Text)
	}
	if !slices.Equal(posts[0].UpdateIDs, []int{300}) {
		t.Errorf("expected first post update IDs [300], got %v", posts[0].UpdateIDs)
	}

	album := posts[1]
	if album.MediaGroupID != "album-2" {
		t.Errorf("expected album second, got media group ID %q", album.MediaGroupID)
	}
	if album.Caption != "Описание второго альбома" {
		t.Errorf("expected album caption %q, got %q", "Описание второго альбома", album.Caption)
	}
	if !slices.Equal(album.UpdateIDs, []int{301, 302}) {
		t.Errorf("expected album update IDs [301 302], got %v", album.UpdateIDs)
	}
	if len(album.Photos) != 2 {
		t.Fatalf("expected album to contain 2 photos, got %d", len(album.Photos))
	}
	if album.Photos[0].FileID != "large-a" || album.Photos[1].FileID != "large-b" {
		t.Errorf(
			"expected album file IDs [large-a large-b], got [%s %s]",
			album.Photos[0].FileID,
			album.Photos[1].FileID,
		)
	}

	if posts[2].Text != "Последний одиночный пост" {
		t.Errorf("expected last single post third, got text %q", posts[2].Text)
	}
	if !slices.Equal(posts[2].UpdateIDs, []int{303}) {
		t.Errorf("expected last post update IDs [303], got %v", posts[2].UpdateIDs)
	}
}

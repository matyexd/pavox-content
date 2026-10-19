package main

import (
	"testing"
	"time"
)

func albumUpdate(id int, chat int64, album string) Update {
	return Update{UpdateID: id, ChannelPost: ChannelPost{
		MessageID: id, Chat: TelegramChat{ID: chat}, MediaGroupID: album,
		Photo: []PhotoSize{{FileID: "small"}, {FileID: "large"}},
	}}
}

func TestAlbumAcrossPollingResponses(t *testing.T) {
	b := &albumBuffer{}
	now := time.Unix(100, 0)
	first := albumUpdate(1, -10, "album")
	first.ChannelPost.Caption = "Product"
	if got := b.collect([]Update{first}, now); len(got) != 0 {
		t.Fatal("album published too early")
	}
	if got := b.collect([]Update{albumUpdate(2, -10, "album")}, now.Add(time.Second)); len(got) != 0 {
		t.Fatal("second part published too early")
	}
	if got := b.collect(nil, now.Add(2*time.Second)); len(got) != 0 {
		t.Fatal("quiet period must restart on new part")
	}
	posts := groupUpdates(b.collect(nil, now.Add(3*time.Second)))
	if len(posts) != 1 || len(posts[0].Photos) != 2 || posts[0].Caption != "Product" {
		t.Fatalf("incomplete album: %#v", posts)
	}
	if len(b.collect(nil, now.Add(4*time.Second))) != 0 {
		t.Fatal("album emitted twice")
	}
}

func TestAlbumPhotoLimitAndChatIsolation(t *testing.T) {
	b := &albumBuffer{}
	now := time.Unix(100, 0)
	updates := []Update{}
	for id := 1; id <= 8; id++ {
		updates = append(updates, albumUpdate(id, -10, "album"))
	}
	updates = append(updates, albumUpdate(9, -20, "album"))
	b.collect(updates, now)
	b.collect(updates, now.Add(time.Second)) // Duplicate deliveries do not extend the timer.
	posts := groupUpdates(b.collect(nil, now.Add(albumQuietPeriod)))
	if len(posts) != 2 || len(posts[0].Photos) != 5 || len(posts[1].Photos) != 1 {
		t.Fatalf("wrong photos/chats: %#v", posts)
	}
	if len(posts[0].UpdateIDs) != 8 {
		t.Fatal("must retain IDs of omitted photos")
	}
	if posts[0].Photos[0].FileID != "large" {
		t.Fatal("must select largest photo size")
	}
}

func TestAlbumStableIdentity(t *testing.T) {
	a := TelegramPost{ChatID: -10, MediaGroupID: "album", UpdateIDs: []int{1, 2}}
	b := a
	b.UpdateIDs = []int{3}
	keyA, _ := telegramSourceID(a)
	keyB, _ := telegramSourceID(b)
	if keyA != keyB {
		t.Fatal("album identity depends on batch boundaries")
	}
	b.ChatID = -20
	keyB, _ = telegramSourceID(b)
	if keyA == keyB {
		t.Fatal("album identity collides between chats")
	}
}

func TestSinglePostDoesNotWaitForAlbum(t *testing.T) {
	b := &albumBuffer{}
	got := b.collect([]Update{albumUpdate(1, -10, "album"), albumUpdate(2, -10, "")}, time.Now())
	if len(got) != 1 || got[0].UpdateID != 2 {
		t.Fatal("single post should process immediately")
	}
}

func TestPendingAlbumOffsetAndDuplicateSingle(t *testing.T) {
	b := &albumBuffer{}
	now := time.Unix(100, 0)
	updates := []Update{albumUpdate(10, -10, "album"), albumUpdate(11, -10, "")}
	if len(b.collect(updates, now)) != 1 {
		t.Fatal("single should be ready")
	}
	if b.confirmedOffset(12) != 10 {
		t.Fatal("pending album must stay unconfirmed")
	}
	if len(b.collect(updates, now.Add(time.Second))) != 0 {
		t.Fatal("repeated delivery must not publish single twice")
	}
	if len(b.collect(updates, now.Add(albumQuietPeriod))) != 1 {
		t.Fatal("duplicates must not postpone album")
	}
	if b.confirmedOffset(12) != 12 || len(b.seen) != 0 {
		t.Fatal("processed updates should be confirmed and released")
	}
}

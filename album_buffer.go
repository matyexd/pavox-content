package main

import (
	"sort"
	"strconv"
	"time"
)

const albumQuietPeriod = 2 * time.Second
const maxProductPhotos = 5

type pendingAlbum struct {
	updates  map[int]Update
	lastSeen time.Time
}

// Used by the polling loop only; never publish a partial batch immediately.
type albumBuffer struct {
	albums map[string]*pendingAlbum
	seen   map[int]bool
}

func (b *albumBuffer) collect(updates []Update, now time.Time) []Update {
	if b.albums == nil {
		b.albums = make(map[string]*pendingAlbum)
	}
	if b.seen == nil {
		b.seen = make(map[int]bool)
	}
	var ready []Update
	for _, update := range updates {
		if b.seen[update.UpdateID] {
			continue
		}
		b.seen[update.UpdateID] = true
		post := update.ChannelPost
		if post.MediaGroupID == "" {
			ready = append(ready, update)
			continue
		}
		key := strconv.FormatInt(post.Chat.ID, 10) + ":" + post.MediaGroupID
		album := b.albums[key]
		if album == nil {
			album = &pendingAlbum{updates: make(map[int]Update)}
			b.albums[key] = album
		}
		if _, exists := album.updates[update.UpdateID]; !exists {
			album.updates[update.UpdateID] = update
			album.lastSeen = now
		}
	}
	for key, album := range b.albums {
		if now.Sub(album.lastSeen) < albumQuietPeriod {
			continue
		}
		for _, update := range album.updates {
			ready = append(ready, update)
		}
		delete(b.albums, key)
	}
	sort.Slice(ready, func(i, j int) bool { return ready[i].UpdateID < ready[j].UpdateID })
	return ready
}

// Keep pending album updates unconfirmed so a restart can receive them again.
func (b *albumBuffer) confirmedOffset(candidate int) int {
	for _, album := range b.albums {
		for id := range album.updates {
			if id < candidate {
				candidate = id
			}
		}
	}
	for id := range b.seen {
		if id < candidate {
			delete(b.seen, id)
		}
	}
	return candidate
}

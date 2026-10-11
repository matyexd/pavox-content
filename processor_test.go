package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestGroupUpdatesEmptyInput(t *testing.T) {
	posts := groupUpdates(nil)

	if len(posts) != 0 {
		t.Fatalf("expected no posts, got %d", len(posts))
	}
}

func TestProcessUpdateBatchAdvancesOffsetAfterSuccess(t *testing.T) {
	updates := []Update{
		{UpdateID: 105, ChannelPost: ChannelPost{Text: "Первый пост"}},
		{UpdateID: 103, ChannelPost: ChannelPost{Text: "Второй пост"}},
	}
	processedPosts := 0

	nextOffset, err := processUpdateBatch(updates, 100, func(post TelegramPost) error {
		processedPosts++
		return nil
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if processedPosts != 2 {
		t.Errorf("expected 2 processed posts, got %d", processedPosts)
	}
	if nextOffset != 106 {
		t.Errorf("expected next offset 106, got %d", nextOffset)
	}
}

func TestProcessUpdateBatchPreservesOffsetAfterError(t *testing.T) {
	updates := []Update{
		{UpdateID: 200, ChannelPost: ChannelPost{Text: "Первый пост"}},
		{UpdateID: 201, ChannelPost: ChannelPost{Text: "Второй пост"}},
	}
	expectedError := errors.New("processing failed")
	processedPosts := 0

	nextOffset, err := processUpdateBatch(updates, 150, func(post TelegramPost) error {
		processedPosts++
		if processedPosts == 2 {
			return expectedError
		}
		return nil
	})

	if !errors.Is(err, expectedError) {
		t.Fatalf("expected processing error, got %v", err)
	}
	if nextOffset != 150 {
		t.Errorf("expected offset to remain 150, got %d", nextOffset)
	}
}

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

type fakeTelegramFileClient struct {
	file    TelegramFile
	content []byte
}

type fakeProductExtractor struct {
	product ProductExtraction
	err     error
	post    TelegramPost
}

type fakeProductImporter struct {
	post       TelegramPost
	product    ProductExtraction
	imagePaths []string
	called     bool
	err        error
}

func (fake *fakeProductImporter) ImportProduct(
	ctx context.Context,
	post TelegramPost,
	product ProductExtraction,
	imagePaths []string,
) (StrapiProduct, bool, error) {
	fake.called = true
	fake.post = post
	fake.product = product
	fake.imagePaths = append([]string(nil), imagePaths...)
	return StrapiProduct{DocumentID: "product-document-id"}, true, fake.err
}

func (fake *fakeProductExtractor) Extract(
	ctx context.Context,
	post TelegramPost,
) (ProductExtraction, error) {
	fake.post = post
	return fake.product, fake.err
}

func (fake fakeTelegramFileClient) getFile(
	fileID string,
) (TelegramResponse[TelegramFile], error) {
	response := TelegramResponse[TelegramFile]{
		OK:     true,
		Result: fake.file,
	}

	return response, nil
}

func (fake fakeTelegramFileClient) downloadTelegramFile(
	file TelegramFile,
	newFile io.Writer,
) error {
	_, err := newFile.Write(fake.content)
	return err
}

func TestProcessPostDownloadsPhoto(t *testing.T) {
	downloadDir := t.TempDir()

	fake := fakeTelegramFileClient{
		file: TelegramFile{
			FileUniqueID: "photo-123",
		},
		content: []byte("fake image content"),
	}

	post := TelegramPost{
		Caption:      "Описание товара",
		MediaGroupID: "media-group-1",
		Photos: []PhotoSize{
			{FileID: "telegram-file-id"},
		},
		UpdateIDs: []int{101},
	}
	productName := "Твёрдый бальзам"
	extractor := &fakeProductExtractor{
		product: ProductExtraction{
			Name:          &productName,
			CategorySlugs: []string{"category-6"},
		},
	}
	importer := &fakeProductImporter{}

	err := processPost(context.Background(), fake, extractor, importer, post, downloadDir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	entries, err := os.ReadDir(downloadDir)
	if err != nil {
		t.Fatalf("read download directory: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one post directory, got %d", len(entries))
	}
	if !entries[0].IsDir() {
		t.Fatalf("expected %q to be a directory", entries[0].Name())
	}
	if _, err := time.Parse("2006-01-02_15-04-05.000000000", entries[0].Name()); err != nil {
		t.Fatalf("expected date-time directory name, got %q", entries[0].Name())
	}

	postDir := filepath.Join(downloadDir, entries[0].Name())
	filePath := filepath.Join(postDir, "photo-123.jpg")

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}

	expectedContent := "fake image content"
	if string(content) != expectedContent {
		t.Errorf(
			"expected file content %q, got %q",
			expectedContent,
			string(content),
		)
	}

	postJSON, err := os.ReadFile(filepath.Join(postDir, "post.json"))
	if err != nil {
		t.Fatalf("read post JSON: %v", err)
	}

	var savedPost TelegramPost
	if err := json.Unmarshal(postJSON, &savedPost); err != nil {
		t.Fatalf("unmarshal post JSON: %v", err)
	}
	if savedPost.Caption != post.Caption {
		t.Errorf("expected caption %q, got %q", post.Caption, savedPost.Caption)
	}
	if savedPost.MediaGroupID != post.MediaGroupID {
		t.Errorf(
			"expected media group ID %q, got %q",
			post.MediaGroupID,
			savedPost.MediaGroupID,
		)
	}
	if !slices.Equal(savedPost.UpdateIDs, post.UpdateIDs) {
		t.Errorf("expected update IDs %v, got %v", post.UpdateIDs, savedPost.UpdateIDs)
	}

	productJSON, err := os.ReadFile(filepath.Join(postDir, "product.json"))
	if err != nil {
		t.Fatalf("read product JSON: %v", err)
	}

	var savedProduct ProductExtraction
	if err := json.Unmarshal(productJSON, &savedProduct); err != nil {
		t.Fatalf("unmarshal product JSON: %v", err)
	}
	if savedProduct.Name == nil || *savedProduct.Name != productName {
		t.Errorf("expected product name %q, got %#v", productName, savedProduct.Name)
	}
	if !slices.Equal(savedProduct.CategorySlugs, []string{"category-6", "category"}) {
		t.Errorf(
			"expected category slugs [category-6 category], got %v",
			savedProduct.CategorySlugs,
		)
	}
	if extractor.post.Caption != post.Caption {
		t.Errorf(
			"expected extractor caption %q, got %q",
			post.Caption,
			extractor.post.Caption,
		)
	}
	if !importer.called {
		t.Fatal("expected Strapi importer to be called")
	}
	if !slices.Equal(importer.imagePaths, []string{filePath}) {
		t.Errorf("expected importer image paths [%s], got %v", filePath, importer.imagePaths)
	}
	if !slices.Equal(importer.product.CategorySlugs, []string{"category-6", "category"}) {
		t.Errorf("unexpected imported category slugs: %v", importer.product.CategorySlugs)
	}
}

func TestProcessPostRejectsInvalidProductExtraction(t *testing.T) {
	downloadDir := t.TempDir()
	emptyName := "   "
	extractor := &fakeProductExtractor{
		product: ProductExtraction{Name: &emptyName},
	}

	err := processPost(
		context.Background(),
		fakeTelegramFileClient{},
		extractor,
		&fakeProductImporter{},
		TelegramPost{Text: "Некорректный товар"},
		downloadDir,
	)
	if err == nil {
		t.Fatal("processPost returned nil, want validation error")
	}
	if !strings.Contains(err.Error(), "validate product extraction: product name is empty") {
		t.Fatalf("unexpected error: %v", err)
	}

	entries, err := os.ReadDir(downloadDir)
	if err != nil {
		t.Fatalf("read download directory: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one post directory, got %d", len(entries))
	}

	productPath := filepath.Join(downloadDir, entries[0].Name(), "product.json")
	if _, err := os.Stat(productPath); !os.IsNotExist(err) {
		t.Fatalf("product.json must not exist after validation error, stat error: %v", err)
	}
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type TelegramFileClient interface {
	getFile(fileID string) (TelegramResponse[TelegramFile], error)
	downloadTelegramFile(file TelegramFile, newFile io.Writer) error
}

type ProductImporter interface {
	ImportProduct(
		ctx context.Context,
		post TelegramPost,
		product ProductExtraction,
		imagePaths []string,
	) (StrapiProduct, bool, error)
}

func processPost(
	ctx context.Context,
	telegramClient TelegramFileClient,
	productExtractor ProductExtractor,
	productImporter ProductImporter,
	post TelegramPost,
	downloadDir string,
	notifiers ...*PublicationNotifier,
) (processErr error) {
	var notifier *PublicationNotifier
	if len(notifiers) > 0 {
		notifier = notifiers[0]
	}
	stage := "сохранение поста и фотографий"
	notifier.send(ctx, post, "📥 Пост получен", "")
	notifier.send(ctx, post, "⏳ Взят в работу", "")
	defer func() {
		if processErr != nil {
			notifier.send(ctx, post, "❌ Публикация не выполнена", "Этап: "+stage+"\n\n<pre>"+html.EscapeString(notificationError(processErr))+"</pre>\nАвтоповтор отключён. После исправления отправьте пост заново.")
		}
	}()
	fmt.Printf("Caption: %s \n", post.Caption)
	fmt.Printf("Text: %s \n", post.Text)
	fmt.Printf("Count photos: %d \n", len(post.Photos))

	postDir := filepath.Join(
		downloadDir,
		time.Now().Format("2006-01-02_15-04-05.000000000"),
	)
	if err := os.MkdirAll(postDir, 0750); err != nil {
		return fmt.Errorf("create post directory: %w", err)
	}

	postJSON, err := json.MarshalIndent(post, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal post: %w", err)
	}

	postFilePath := filepath.Join(postDir, "post.json")
	if err := os.WriteFile(postFilePath, postJSON, 0640); err != nil {
		return fmt.Errorf("write post JSON: %w", err)
	}

	imagePaths := make([]string, 0, len(post.Photos))
	for index, photo := range post.Photos {
		fmt.Printf("#%d \n", index+1)
		fmt.Printf("Width: %d \n", photo.Width)
		fmt.Printf("Height: %d \n", photo.Height)

		file, errorGetFile := telegramClient.getFile(photo.FileID)
		if errorGetFile != nil {
			return fmt.Errorf("get file: %w", errorGetFile)
		}

		fileName := file.Result.FileUniqueID + ".jpg"
		fileWithDir := filepath.Join(postDir, fileName)

		newFile, errorCreate := os.Create(fileWithDir)
		if errorCreate != nil {
			return fmt.Errorf("create file: %w", errorCreate)
		}

		downloadError := telegramClient.downloadTelegramFile(file.Result, newFile)
		closeError := newFile.Close()

		if downloadError != nil {
			return fmt.Errorf("download file: %w", downloadError)
		}

		if closeError != nil {
			return fmt.Errorf("close file: %w", closeError)
		}

		imagePaths = append(imagePaths, fileWithDir)
	}

	stage = "извлечение и проверка товара"
	product, err := productExtractor.Extract(ctx, post)
	if err != nil {
		return fmt.Errorf("extract product with LLM: %w", err)
	}
	if err := validateProductExtraction(product); err != nil {
		return fmt.Errorf("validate product extraction: %w", err)
	}

	product.CategorySlugs = addParentCategorySlugs(product.CategorySlugs)

	productJSON, err := json.MarshalIndent(product, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal product: %w", err)
	}

	productFilePath := filepath.Join(postDir, "product.json")
	if err := os.WriteFile(productFilePath, productJSON, 0640); err != nil {
		return fmt.Errorf("write product JSON: %w", err)
	}

	if product.Name == nil {
		notifier.send(ctx, post, "⏭ Пост пропущен", "В посте не найден товар.")
		fmt.Println("Publication does not contain a product; skipping Strapi import")
		return nil
	}

	stage = "публикация в Strapi"
	published, created, err := productImporter.ImportProduct(ctx, post, product, imagePaths)
	if err != nil {
		return fmt.Errorf("import product to Strapi: %w", err)
	}
	notifier.published(ctx, post, published)
	if created {
		fmt.Printf("Created and published Strapi product %s\n", published.DocumentID)
	} else {
		fmt.Printf("Published existing Strapi product %s\n", published.DocumentID)
	}

	return nil
}

func groupUpdates(updates []Update) []TelegramPost {
	posts := make(map[string][]Update)
	order := make([]string, 0)

	telegramPosts := make([]TelegramPost, 0)

	for _, post := range updates {
		if post.ChannelPost.MessageID == 0 && post.ChannelPost.Text == "" && post.ChannelPost.Caption == "" && len(post.ChannelPost.Photo) == 0 {
			continue
		}
		var key string
		if post.ChannelPost.MediaGroupID == "" {
			key = "single:" + strconv.Itoa(post.UpdateID)
		} else {
			key = "album:" + strconv.FormatInt(post.ChannelPost.Chat.ID, 10) + ":" + post.ChannelPost.MediaGroupID
		}

		if _, exists := posts[key]; !exists {
			order = append(order, key)
		}

		posts[key] = append(posts[key], post)
	}

	for _, key := range order {
		var telegramPost TelegramPost
		groupPosts := posts[key]
		telegramPost.ChatID = groupPosts[0].ChannelPost.Chat.ID
		telegramPost.MediaGroupID = groupPosts[0].ChannelPost.MediaGroupID

		for _, post := range groupPosts {
			if len(post.ChannelPost.Photo) > 0 {
				telegramPost.Photos = append(telegramPost.Photos, post.ChannelPost.Photo[len(post.ChannelPost.Photo)-1])
			}
			telegramPost.UpdateIDs = append(telegramPost.UpdateIDs, post.UpdateID)
		}

		for _, group := range groupPosts {
			if group.ChannelPost.Caption != "" {
				telegramPost.Caption = group.ChannelPost.Caption
				break
			}
		}

		for _, group := range groupPosts {
			if group.ChannelPost.Text != "" {
				telegramPost.Text = group.ChannelPost.Text
				break
			}
		}

		telegramPosts = append(telegramPosts, telegramPost)
	}

	return telegramPosts
}

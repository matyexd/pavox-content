package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type StrapiMedia struct {
	ID         int    `json:"id"`
	DocumentID string `json:"documentId"`
	Name       string `json:"name"`
	MIME       string `json:"mime"`
	URL        string `json:"url"`
}

func (client *StrapiClient) UploadFiles(
	ctx context.Context,
	filePaths []string,
) ([]StrapiMedia, error) {
	uploaded := make([]StrapiMedia, 0, len(filePaths))
	for _, filePath := range filePaths {
		media, err := client.uploadFile(ctx, filePath)
		if err != nil {
			cleanupErr := client.deleteMediaFiles(ctx, uploaded)
			return nil, errors.Join(err, cleanupErr)
		}
		uploaded = append(uploaded, media)
	}

	return uploaded, nil
}

func (client *StrapiClient) uploadFile(
	ctx context.Context,
	filePath string,
) (StrapiMedia, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return StrapiMedia{}, fmt.Errorf("open upload file %q: %w", filePath, err)
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fileName := filepath.Base(filePath)
	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(fileName)))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{
		"name":     "files",
		"filename": fileName,
	}))
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		return StrapiMedia{}, fmt.Errorf("create upload form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return StrapiMedia{}, fmt.Errorf("copy upload file %q: %w", filePath, err)
	}
	if err := writer.Close(); err != nil {
		return StrapiMedia{}, fmt.Errorf("close upload multipart body: %w", err)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		client.baseURL+"/api/upload",
		&body,
	)
	if err != nil {
		return StrapiMedia{}, fmt.Errorf("create upload request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+client.apiToken)
	request.Header.Set("Content-Type", writer.FormDataContentType())

	response, err := client.httpClient.Do(request)
	if err != nil {
		return StrapiMedia{}, fmt.Errorf("upload file %q: %w", filePath, err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		errorBody, readErr := io.ReadAll(io.LimitReader(response.Body, 16*1024))
		if readErr != nil {
			return StrapiMedia{}, fmt.Errorf(
				"Strapi upload returned %s; read error body: %w",
				response.Status,
				readErr,
			)
		}
		return StrapiMedia{}, fmt.Errorf(
			"Strapi upload returned %s: %s",
			response.Status,
			strings.TrimSpace(string(errorBody)),
		)
	}

	var uploaded []StrapiMedia
	if err := json.NewDecoder(response.Body).Decode(&uploaded); err != nil {
		return StrapiMedia{}, fmt.Errorf("decode Strapi upload response: %w", err)
	}
	if len(uploaded) != 1 || uploaded[0].ID == 0 {
		return StrapiMedia{}, fmt.Errorf(
			"unexpected Strapi upload response: got %d files",
			len(uploaded),
		)
	}

	return uploaded[0], nil
}

func (client *StrapiClient) DeleteMedia(
	ctx context.Context,
	mediaID int,
) error {
	if mediaID <= 0 {
		return fmt.Errorf("Strapi media ID must be positive")
	}

	path := "/api/upload/files/" + url.PathEscape(fmt.Sprint(mediaID))
	if err := client.doJSON(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return fmt.Errorf("delete Strapi media %d: %w", mediaID, err)
	}

	return nil
}

func (client *StrapiClient) deleteMediaFiles(
	ctx context.Context,
	mediaFiles []StrapiMedia,
) error {
	var cleanupErrors []error
	for _, media := range mediaFiles {
		if err := client.DeleteMedia(ctx, media.ID); err != nil {
			cleanupErrors = append(cleanupErrors, err)
		}
	}

	return errors.Join(cleanupErrors...)
}

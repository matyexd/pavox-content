package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestStrapiClientUploadFiles(t *testing.T) {
	imagePath := filepath.Join(t.TempDir(), "product.jpg")
	if err := os.WriteFile(imagePath, []byte("image bytes"), 0600); err != nil {
		t.Fatalf("write test image: %v", err)
	}

	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests++
		if request.Header.Get("Authorization") != "Bearer token" {
			t.Errorf("Authorization header = %q", request.Header.Get("Authorization"))
		}

		switch requests {
		case 1:
			if request.Method != http.MethodPost || request.URL.Path != "/api/upload" {
				t.Fatalf("upload request = %s %s", request.Method, request.URL.String())
			}
			file, header, err := request.FormFile("files")
			if err != nil {
				t.Fatalf("read multipart file: %v", err)
			}
			defer file.Close()
			content, err := io.ReadAll(file)
			if err != nil {
				t.Fatalf("read uploaded content: %v", err)
			}
			if header.Filename != "product.jpg" || string(content) != "image bytes" {
				t.Errorf("uploaded file = %q %q", header.Filename, content)
			}
			if got := header.Header.Get("Content-Type"); got != "image/jpeg" {
				t.Errorf("uploaded Content-Type = %q, want image/jpeg", got)
			}

			writeJSON(t, response, http.StatusCreated, []map[string]any{{
				"id":         119,
				"documentId": "media-document-id",
				"name":       "product.jpg",
				"mime":       "image/jpeg",
				"url":        "/uploads/product.jpg",
			}})
		case 2:
			if request.Method != http.MethodDelete || request.URL.Path != "/api/upload/files/119" {
				t.Fatalf("delete request = %s %s", request.Method, request.URL.String())
			}
			writeJSON(t, response, http.StatusOK, map[string]any{"id": 119})
		default:
			t.Fatalf("unexpected request #%d", requests)
		}
	}))
	defer server.Close()

	client, err := NewStrapiClient(server.URL, "token", server.Client())
	if err != nil {
		t.Fatalf("NewStrapiClient returned error: %v", err)
	}

	media, err := client.UploadFiles(context.Background(), []string{imagePath})
	if err != nil {
		t.Fatalf("UploadFiles returned error: %v", err)
	}
	if len(media) != 1 || media[0].ID != 119 {
		t.Fatalf("unexpected uploaded media: %#v", media)
	}

	if err := client.DeleteMedia(context.Background(), media[0].ID); err != nil {
		t.Fatalf("DeleteMedia returned error: %v", err)
	}
	if requests != 2 {
		t.Fatalf("request count = %d, want 2", requests)
	}
}

package main

import (
	"context"
	"encoding/base64"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStrapiUploadIntegration(t *testing.T) {
	if os.Getenv("STRAPI_INTEGRATION") != "1" {
		t.Skip("set STRAPI_INTEGRATION=1 to run against a local Strapi")
	}

	baseURL := os.Getenv("STRAPI_BASE_URL")
	apiToken := os.Getenv("STRAPI_API_TOKEN")
	client, err := NewStrapiClient(
		baseURL,
		apiToken,
		&http.Client{Timeout: 30 * time.Second},
	)
	if err != nil {
		t.Fatalf("NewStrapiClient returned error: %v", err)
	}

	image, err := base64.StdEncoding.DecodeString(
		"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=",
	)
	if err != nil {
		t.Fatalf("decode test PNG: %v", err)
	}
	imagePath := filepath.Join(t.TempDir(), "strapi-upload-integration.png")
	if err := os.WriteFile(imagePath, image, 0600); err != nil {
		t.Fatalf("write test PNG: %v", err)
	}

	uploaded, err := client.UploadFiles(context.Background(), []string{imagePath})
	if err != nil {
		t.Fatalf("UploadFiles returned error: %v", err)
	}
	if len(uploaded) != 1 {
		t.Fatalf("uploaded file count = %d, want 1", len(uploaded))
	}
	t.Cleanup(func() {
		if err := client.DeleteMedia(context.Background(), uploaded[0].ID); err != nil {
			t.Errorf("delete integration test media: %v", err)
		}
	})

	if uploaded[0].MIME != "image/png" {
		t.Errorf("uploaded MIME = %q, want image/png", uploaded[0].MIME)
	}
	if uploaded[0].URL == "" {
		t.Error("uploaded URL is empty")
	}
}

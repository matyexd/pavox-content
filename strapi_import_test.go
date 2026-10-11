package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestProductSlug(t *testing.T) {
	got := productSlug("Твёрдый бальзам №1", []int{102, 100, 100})
	want := "tvyordyy-balzam-1-100-102"
	if got != want {
		t.Fatalf("slug = %q, want %q", got, want)
	}
}

func TestStrapiProductInput(t *testing.T) {
	name := "  Твёрдый бальзам  "
	description := "Описание"
	price := 900
	weight := 50

	input := strapiProductInput(
		TelegramPost{UpdateIDs: []int{101}},
		ProductExtraction{
			Name:          &name,
			Description:   &description,
			Price:         &price,
			WeightGrams:   &weight,
			CategorySlugs: []string{"category-6", "category"},
		},
		"telegram:101",
		[]int{143, 128},
		[]StrapiMedia{{ID: 119}, {ID: 120}},
	)

	if input.Name != "Твёрдый бальзам" {
		t.Errorf("name = %q", input.Name)
	}
	if input.Slug != "tvyordyy-balzam-101" {
		t.Errorf("slug = %q", input.Slug)
	}
	if input.Price == nil || input.Price.Price != 900 || input.Price.CurrentPrice != 900 || input.Price.Currency != "RUB" {
		t.Errorf("price = %#v", input.Price)
	}
	if !reflect.DeepEqual(input.Categories, []int{143, 128}) {
		t.Errorf("categories = %v", input.Categories)
	}
	if !reflect.DeepEqual(input.Media, []int{119, 120}) {
		t.Errorf("media = %v", input.Media)
	}
}

func TestImportProductPublishesExistingDraftWithoutUploadingAgain(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests++
		switch requests {
		case 1:
			if request.Method != http.MethodGet || request.URL.Path != "/api/products" {
				t.Fatalf("find request = %s %s", request.Method, request.URL.String())
			}
			writeJSON(t, response, http.StatusOK, map[string]any{
				"data": []map[string]any{{
					"id":               200,
					"documentId":       "existing-document-id",
					"telegramSourceId": "telegram:101",
				}},
			})
		case 2:
			if request.Method != http.MethodPut || request.URL.Path != "/api/products/existing-document-id" {
				t.Fatalf("publish request = %s %s", request.Method, request.URL.String())
			}
			writeJSON(t, response, http.StatusOK, map[string]any{
				"data": map[string]any{
					"id":          201,
					"documentId":  "existing-document-id",
					"publishedAt": "2026-10-12T00:00:00.000Z",
				},
			})
		default:
			t.Fatalf("unexpected request #%d", requests)
		}
	}))
	defer server.Close()

	client, err := NewStrapiClient(server.URL, "token", server.Client())
	if err != nil {
		t.Fatalf("NewStrapiClient returned error: %v", err)
	}
	name := "Товар"
	published, created, err := client.ImportProduct(
		context.Background(),
		TelegramPost{UpdateIDs: []int{101}},
		ProductExtraction{Name: &name},
		[]string{"file-that-must-not-be-opened.jpg"},
	)
	if err != nil {
		t.Fatalf("ImportProduct returned error: %v", err)
	}
	if created {
		t.Fatal("created = true, want false")
	}
	if published.DocumentID != "existing-document-id" || published.PublishedAt == nil {
		t.Fatalf("unexpected published product: %#v", published)
	}
	if requests != 2 {
		t.Fatalf("request count = %d, want 2", requests)
	}
}

func TestImportProductCreatesAndPublishesProduct(t *testing.T) {
	name := "Твёрдый бальзам"
	price := 900
	requests := 0

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests++
		switch requests {
		case 1:
			writeJSON(t, response, http.StatusOK, map[string]any{"data": []any{}})
		case 2:
			writeJSON(t, response, http.StatusOK, map[string]any{
				"data": []map[string]any{{"id": 143, "slug": "category-6"}},
			})
		case 3:
			var body struct {
				Data StrapiProductInput `json:"data"`
			}
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Fatalf("decode product request: %v", err)
			}
			if body.Data.TelegramSourceID != "telegram:101" || body.Data.Slug != "tvyordyy-balzam-101" {
				t.Errorf("unexpected product input: %#v", body.Data)
			}
			writeJSON(t, response, http.StatusCreated, map[string]any{
				"data": map[string]any{"id": 200, "documentId": "new-document-id"},
			})
		case 4:
			writeJSON(t, response, http.StatusOK, map[string]any{
				"data": map[string]any{
					"id":          201,
					"documentId":  "new-document-id",
					"publishedAt": "2026-10-12T00:00:00.000Z",
				},
			})
		default:
			t.Fatalf("unexpected request #%d: %s %s", requests, request.Method, request.URL.String())
		}
	}))
	defer server.Close()

	client, err := NewStrapiClient(server.URL, "token", server.Client())
	if err != nil {
		t.Fatalf("NewStrapiClient returned error: %v", err)
	}
	published, created, err := client.ImportProduct(
		context.Background(),
		TelegramPost{UpdateIDs: []int{101}},
		ProductExtraction{
			Name:          &name,
			Price:         &price,
			CategorySlugs: []string{"category-6"},
		},
		nil,
	)
	if err != nil {
		t.Fatalf("ImportProduct returned error: %v", err)
	}
	if !created || published.DocumentID != "new-document-id" || published.PublishedAt == nil {
		t.Fatalf("unexpected import result: created=%v product=%#v", created, published)
	}
	if requests != 4 {
		t.Fatalf("request count = %d, want 4", requests)
	}
}

func TestImportProductDeletesUploadedMediaWhenDraftCreationFails(t *testing.T) {
	imagePath := filepath.Join(t.TempDir(), "product.jpg")
	if err := os.WriteFile(imagePath, []byte("image bytes"), 0600); err != nil {
		t.Fatalf("write test image: %v", err)
	}

	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests++
		switch requests {
		case 1:
			writeJSON(t, response, http.StatusOK, map[string]any{"data": []any{}})
		case 2:
			writeJSON(t, response, http.StatusOK, map[string]any{
				"data": []map[string]any{{"id": 143, "slug": "category-6"}},
			})
		case 3:
			if request.Method != http.MethodPost || request.URL.Path != "/api/upload" {
				t.Fatalf("upload request = %s %s", request.Method, request.URL.String())
			}
			writeJSON(t, response, http.StatusCreated, []map[string]any{{"id": 119}})
		case 4:
			if request.Method != http.MethodPost || request.URL.Path != "/api/products" {
				t.Fatalf("create request = %s %s", request.Method, request.URL.String())
			}
			writeJSON(t, response, http.StatusInternalServerError, map[string]any{
				"error": map[string]any{"message": "create failed"},
			})
		case 5:
			if request.Method != http.MethodDelete || request.URL.Path != "/api/upload/files/119" {
				t.Fatalf("cleanup request = %s %s", request.Method, request.URL.String())
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
	name := "Товар"
	_, _, err = client.ImportProduct(
		context.Background(),
		TelegramPost{UpdateIDs: []int{101}},
		ProductExtraction{Name: &name, CategorySlugs: []string{"category-6"}},
		[]string{imagePath},
	)
	if err == nil {
		t.Fatal("ImportProduct returned nil, want error")
	}
	if requests != 5 {
		t.Fatalf("request count = %d, want 5", requests)
	}
}

package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestStrapiClientProductLifecycle(t *testing.T) {
	const apiToken = "test-token"
	requests := 0

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests++
		if request.Header.Get("Authorization") != "Bearer "+apiToken {
			t.Errorf("Authorization header = %q", request.Header.Get("Authorization"))
		}

		switch requests {
		case 1:
			if request.Method != http.MethodGet || request.URL.Path != "/api/categories" {
				t.Fatalf("first request = %s %s", request.Method, request.URL.String())
			}
			writeJSON(t, response, http.StatusOK, map[string]any{
				"data": []map[string]any{
					{"id": 143, "name": "Бальзамы", "slug": "category-6"},
					{"id": 128, "name": "Для волос", "slug": "category"},
				},
			})
		case 2:
			if request.Method != http.MethodPost || request.URL.Path != "/api/products" {
				t.Fatalf("second request = %s %s", request.Method, request.URL.String())
			}
			if request.URL.Query().Get("status") != "draft" {
				t.Errorf("create status = %q, want draft", request.URL.Query().Get("status"))
			}

			var body struct {
				Data StrapiProductInput `json:"data"`
			}
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Fatalf("decode create request: %v", err)
			}
			if body.Data.Name != "Твёрдый бальзам" {
				t.Errorf("product name = %q", body.Data.Name)
			}
			if body.Data.TelegramSourceID != "telegram:100,101" {
				t.Errorf("Telegram source ID = %q", body.Data.TelegramSourceID)
			}
			if !reflect.DeepEqual(body.Data.Categories, []int{143, 128}) {
				t.Errorf("categories = %v", body.Data.Categories)
			}

			writeJSON(t, response, http.StatusCreated, map[string]any{
				"data": map[string]any{
					"id":          200,
					"documentId":  "product-document-id",
					"name":        body.Data.Name,
					"slug":        body.Data.Slug,
					"publishedAt": nil,
				},
			})
		case 3:
			if request.Method != http.MethodPut || request.URL.Path != "/api/products/product-document-id" {
				t.Fatalf("third request = %s %s", request.Method, request.URL.String())
			}

			var body struct {
				Data map[string]any `json:"data"`
			}
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Fatalf("decode publish request: %v", err)
			}
			if len(body.Data) != 0 {
				t.Errorf("publish data = %#v, want empty object", body.Data)
			}

			writeJSON(t, response, http.StatusOK, map[string]any{
				"data": map[string]any{
					"id":          201,
					"documentId":  "product-document-id",
					"name":        "Твёрдый бальзам",
					"slug":        "tvyordyy-balzam-101",
					"publishedAt": "2026-10-11T16:00:00.000Z",
				},
			})
		default:
			t.Fatalf("unexpected request #%d", requests)
		}
	}))
	defer server.Close()

	client, err := NewStrapiClient(server.URL, apiToken, server.Client())
	if err != nil {
		t.Fatalf("NewStrapiClient returned error: %v", err)
	}

	categoryIDs, err := client.CategoryIDsBySlugs(
		context.Background(),
		[]string{"category-6", "category"},
	)
	if err != nil {
		t.Fatalf("CategoryIDsBySlugs returned error: %v", err)
	}
	if !reflect.DeepEqual(categoryIDs, []int{143, 128}) {
		t.Fatalf("category IDs = %v", categoryIDs)
	}

	draft, err := client.CreateProductDraft(context.Background(), StrapiProductInput{
		Name:             "Твёрдый бальзам",
		Slug:             "tvyordyy-balzam-101",
		TelegramSourceID: "telegram:100,101",
		Categories:       categoryIDs,
	})
	if err != nil {
		t.Fatalf("CreateProductDraft returned error: %v", err)
	}
	if draft.DocumentID != "product-document-id" || draft.PublishedAt != nil {
		t.Fatalf("unexpected draft: %#v", draft)
	}

	published, err := client.PublishProduct(context.Background(), draft.DocumentID)
	if err != nil {
		t.Fatalf("PublishProduct returned error: %v", err)
	}
	if published.PublishedAt == nil {
		t.Fatal("published product has nil publishedAt")
	}
	if requests != 3 {
		t.Fatalf("request count = %d, want 3", requests)
	}
}

func TestTelegramSourceID(t *testing.T) {
	sourceID, err := telegramSourceID(TelegramPost{UpdateIDs: []int{102, 100, 101, 100}})
	if err != nil {
		t.Fatalf("telegramSourceID returned error: %v", err)
	}
	if sourceID != "telegram:100,101,102" {
		t.Fatalf("source ID = %q", sourceID)
	}
}

func TestCreateProductDraftRequiresTelegramSourceID(t *testing.T) {
	client, err := NewStrapiClient("http://strapi.test", "token", http.DefaultClient)
	if err != nil {
		t.Fatalf("NewStrapiClient returned error: %v", err)
	}

	_, err = client.CreateProductDraft(context.Background(), StrapiProductInput{
		Name: "Товар",
		Slug: "tovar",
	})
	if err == nil {
		t.Fatal("CreateProductDraft returned nil, want error")
	}
}

func TestTelegramSourceIDRequiresUpdateID(t *testing.T) {
	_, err := telegramSourceID(TelegramPost{})
	if err == nil {
		t.Fatal("telegramSourceID returned nil, want error")
	}
}

func TestStrapiClientFindProductByTelegramSourceID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Path != "/api/products" {
			t.Fatalf("request = %s %s", request.Method, request.URL.String())
		}
		if got := request.URL.Query().Get("status"); got != "draft" {
			t.Errorf("status = %q", got)
		}
		if got := request.URL.Query().Get("filters[telegramSourceId][$eq]"); got != "telegram:100,101" {
			t.Errorf("Telegram source ID filter = %q", got)
		}

		writeJSON(t, response, http.StatusOK, map[string]any{
			"data": []map[string]any{{
				"id":               200,
				"documentId":       "product-document-id",
				"name":             "Твёрдый бальзам",
				"slug":             "tvyordyy-balzam-101",
				"telegramSourceId": "telegram:100,101",
				"publishedAt":      nil,
			}},
		})
	}))
	defer server.Close()

	client, err := NewStrapiClient(server.URL, "token", server.Client())
	if err != nil {
		t.Fatalf("NewStrapiClient returned error: %v", err)
	}

	product, err := client.FindProductByTelegramSourceID(context.Background(), "telegram:100,101")
	if err != nil {
		t.Fatalf("FindProductByTelegramSourceID returned error: %v", err)
	}
	if product == nil || product.DocumentID != "product-document-id" {
		t.Fatalf("unexpected product: %#v", product)
	}
}

func TestStrapiClientCategoryNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		writeJSON(t, response, http.StatusOK, map[string]any{"data": []any{}})
	}))
	defer server.Close()

	client, err := NewStrapiClient(server.URL, "token", server.Client())
	if err != nil {
		t.Fatalf("NewStrapiClient returned error: %v", err)
	}

	_, err = client.CategoryIDsBySlugs(context.Background(), []string{"unknown"})
	if err == nil {
		t.Fatal("CategoryIDsBySlugs returned nil, want error")
	}
}

func writeJSON(t *testing.T, response http.ResponseWriter, status int, value any) {
	t.Helper()
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	if err := json.NewEncoder(response).Encode(value); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

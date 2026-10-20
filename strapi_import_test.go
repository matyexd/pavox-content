package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProductSlug(t *testing.T) {
	if got := productSlug("Твёрдый бальзам №1", []int{102, 100, 100}); got != "tvyordyy-balzam-1-100-102" {
		t.Fatal(got)
	}
}

func TestImportProductWritesSizesAndPricesToSKU(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(map[bool]string{false: "new", true: "existing"}[existing], func(t *testing.T) {
			name := "Крем"
			small, large, grams, p700, p1200, p400 := 50, 100, 300, 700, 1200, 400
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				if r.Method == http.MethodGet && r.URL.Path == "/api/products" {
					rows := []map[string]any{}
					if existing {
						rows = append(rows, map[string]any{"documentId": "old"})
					}
					writeJSON(t, w, 200, map[string]any{"data": rows})
					return
				}
				if r.Method != http.MethodPost || r.URL.Path != "/api/internal/catalog/telegram-import" {
					t.Errorf("unexpected %s %s", r.Method, r.URL)
					w.WriteHeader(500)
					return
				}
				var body struct {
					Data StrapiProductInput `json:"data"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if body.Data.TelegramSourceID != "telegram:101" || len(body.Data.Variants) != 3 {
					t.Fatalf("bad input: %#v", body.Data)
				}
				a, b, c := body.Data.Variants[0], body.Data.Variants[1], body.Data.Variants[2]
				if a.WeightGrams != nil || *a.VolumeMl != 50 || *a.Price != 700 || *b.VolumeMl != 100 || *b.Price != 1200 || *c.WeightGrams != 300 || c.VolumeMl != nil || *c.Price != 400 {
					t.Fatalf("wrong units/prices: %#v", body.Data.Variants)
				}
				writeJSON(t, w, 200, map[string]any{"data": map[string]any{"documentId": "canonical", "publishedAt": "2026-10-20"}, "created": !existing})
			}))
			defer server.Close()
			client, _ := NewStrapiClient(server.URL, "token", server.Client())
			paths := []string(nil)
			if existing {
				paths = []string{"must-not-be-uploaded.jpg"}
			}
			result, created, err := client.ImportProduct(context.Background(), TelegramPost{UpdateIDs: []int{101}}, ProductExtraction{Name: &name, Variants: []ProductVariantExtraction{
				{VolumeMl: &small, Price: &p700}, {VolumeMl: &large, Price: &p1200}, {WeightGrams: &grams, Price: &p400},
			}}, paths)
			if err != nil || created == existing || result.DocumentID != "canonical" || requests != 2 {
				t.Fatalf("result=%#v created=%v err=%v requests=%d", result, created, err, requests)
			}
		})
	}
}

func TestImportProductDoesNotFallBackToLegacyWrites(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			writeJSON(t, w, 200, map[string]any{"data": []any{}})
			return
		}
		if r.URL.Path != "/api/internal/catalog/telegram-import" {
			t.Errorf("legacy write: %s", r.URL)
		}
		w.WriteHeader(404)
	}))
	defer server.Close()
	client, _ := NewStrapiClient(server.URL, "token", server.Client())
	name := "Товар"
	_, _, err := client.ImportProduct(context.Background(), TelegramPost{UpdateIDs: []int{101}}, ProductExtraction{Name: &name}, nil)
	if err == nil {
		t.Fatal("missing backend endpoint must fail")
	}
}

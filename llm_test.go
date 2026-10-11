package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestProductExtractionRequest(t *testing.T) {
	post := TelegramPost{
		Caption:      "Описание товара",
		Text:         "Дополнительный текст",
		MediaGroupID: "media-group",
		Photos: []PhotoSize{
			{FileID: "secret-file-id"},
		},
		UpdateIDs: []int{123},
	}

	request, err := productExtractionRequest(post)
	if err != nil {
		t.Fatalf("productExtractionRequest returned error: %v", err)
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	var body struct {
		Model string `json:"model"`
		Input []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"input"`
		Store bool `json:"store"`
		Text  struct {
			Format struct {
				Strict bool           `json:"strict"`
				Schema map[string]any `json:"schema"`
			} `json:"format"`
		} `json:"text"`
	}
	if err := json.Unmarshal(requestJSON, &body); err != nil {
		t.Fatalf("decode request: %v", err)
	}

	if body.Model != "gpt-6-luna" {
		t.Errorf("model = %q, want gpt-6-luna", body.Model)
	}
	if len(body.Input) != 2 {
		t.Fatalf("input message count = %d, want 2", len(body.Input))
	}
	if body.Input[0].Role != "developer" {
		t.Errorf("first message role = %q, want developer", body.Input[0].Role)
	}
	if body.Input[1].Role != "user" {
		t.Errorf("second message role = %q, want user", body.Input[1].Role)
	}
	if !body.Text.Format.Strict {
		t.Error("text.format.strict = false, want true")
	}
	if body.Store {
		t.Error("store = true, want false")
	}

	var input map[string]any
	if err := json.Unmarshal([]byte(body.Input[1].Content), &input); err != nil {
		t.Fatalf("decode user message: %v", err)
	}
	if len(input) != 2 || input["caption"] != post.Caption || input["text"] != post.Text {
		t.Errorf("user message = %#v, want only caption and text", input)
	}

	properties := body.Text.Format.Schema["properties"].(map[string]any)
	nameSchema := properties["name"].(map[string]any)
	if !reflect.DeepEqual(nameSchema["type"], []any{"string", "null"}) {
		t.Errorf("name type = %#v, want [string null]", nameSchema["type"])
	}

	categorySchema := properties["category_slugs"].(map[string]any)
	items := categorySchema["items"].(map[string]any)
	wantSlugs := make([]any, 0, len(productCategories))
	for _, slug := range productCategorySlugs() {
		wantSlugs = append(wantSlugs, slug)
	}
	if !reflect.DeepEqual(items["enum"], wantSlugs) {
		t.Errorf("category enum = %#v, want %#v", items["enum"], wantSlugs)
	}
}

func TestResponseLLMHandle(t *testing.T) {
	responseJSON := `{
		"name": "Твёрдый бальзам",
		"description": null,
		"compound": "Воск розы и воск жасмина",
		"method_application": null,
		"price": null,
		"weight_grams": null,
		"category_slugs": ["category-6"]
	}`

	extraction, err := responseLLMHandle(responseJSON)
	if err != nil {
		t.Fatalf("responseLLMHandle returned error: %v", err)
	}

	if extraction.Name == nil || *extraction.Name != "Твёрдый бальзам" {
		t.Errorf("name = %#v, want Твёрдый бальзам", extraction.Name)
	}
	if extraction.MethodApplication != nil {
		t.Errorf("method application = %#v, want nil", extraction.MethodApplication)
	}
	if !reflect.DeepEqual(extraction.CategorySlugs, []string{"category-6"}) {
		t.Errorf("category slugs = %#v, want [category-6]", extraction.CategorySlugs)
	}
}

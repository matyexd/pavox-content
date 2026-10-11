package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

type StrapiClient struct {
	baseURL    string
	apiToken   string
	httpClient *http.Client
}

type StrapiCategory struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type StrapiProductInput struct {
	Name              string       `json:"name"`
	Slug              string       `json:"slug"`
	TelegramSourceID  string       `json:"telegramSourceId"`
	Description       *string      `json:"description,omitempty"`
	Compound          *string      `json:"compound,omitempty"`
	MethodApplication *string      `json:"method_application,omitempty"`
	Price             *StrapiPrice `json:"price,omitempty"`
	WeightGrams       *int         `json:"weightGrams,omitempty"`
	Categories        []int        `json:"categories,omitempty"`
	Media             []int        `json:"media,omitempty"`
}

type StrapiPrice struct {
	Price        int    `json:"price"`
	CurrentPrice int    `json:"currentPrice"`
	Currency     string `json:"currency"`
}

type StrapiProduct struct {
	ID               int     `json:"id"`
	DocumentID       string  `json:"documentId"`
	Name             string  `json:"name"`
	Slug             string  `json:"slug"`
	TelegramSourceID string  `json:"telegramSourceId"`
	PublishedAt      *string `json:"publishedAt"`
}

func telegramSourceID(post TelegramPost) (string, error) {
	if len(post.UpdateIDs) == 0 {
		return "", fmt.Errorf("Telegram post has no update IDs")
	}

	updateIDs := append([]int(nil), post.UpdateIDs...)
	sort.Ints(updateIDs)

	parts := make([]string, 0, len(updateIDs))
	for index, updateID := range updateIDs {
		if index > 0 && updateID == updateIDs[index-1] {
			continue
		}
		parts = append(parts, strconv.Itoa(updateID))
	}

	return "telegram:" + strings.Join(parts, ","), nil
}

func NewStrapiClient(
	baseURL string,
	apiToken string,
	httpClient *http.Client,
) (*StrapiClient, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("Strapi base URL is empty")
	}
	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return nil, fmt.Errorf("parse Strapi base URL: %w", err)
	}
	if strings.TrimSpace(apiToken) == "" {
		return nil, fmt.Errorf("Strapi API token is empty")
	}
	if httpClient == nil {
		return nil, fmt.Errorf("Strapi HTTP client is nil")
	}

	return &StrapiClient{
		baseURL:    baseURL,
		apiToken:   apiToken,
		httpClient: httpClient,
	}, nil
}

func (client *StrapiClient) CategoryIDsBySlugs(
	ctx context.Context,
	slugs []string,
) ([]int, error) {
	if len(slugs) == 0 {
		return []int{}, nil
	}

	var response struct {
		Data []StrapiCategory `json:"data"`
	}
	if err := client.doJSON(ctx, http.MethodGet, "/api/categories", nil, &response); err != nil {
		return nil, fmt.Errorf("get Strapi categories: %w", err)
	}

	categoriesBySlug := make(map[string]StrapiCategory, len(response.Data))
	for _, category := range response.Data {
		categoriesBySlug[category.Slug] = category
	}

	ids := make([]int, 0, len(slugs))
	for _, slug := range slugs {
		category, exists := categoriesBySlug[slug]
		if !exists {
			return nil, fmt.Errorf("Strapi category not found: %q", slug)
		}
		ids = append(ids, category.ID)
	}

	return ids, nil
}

func (client *StrapiClient) CreateProductDraft(
	ctx context.Context,
	product StrapiProductInput,
) (StrapiProduct, error) {
	if strings.TrimSpace(product.TelegramSourceID) == "" {
		return StrapiProduct{}, fmt.Errorf("create Strapi product draft: Telegram source ID is empty")
	}

	body := struct {
		Data StrapiProductInput `json:"data"`
	}{Data: product}

	var response struct {
		Data StrapiProduct `json:"data"`
	}
	if err := client.doJSON(
		ctx,
		http.MethodPost,
		"/api/products?status=draft",
		body,
		&response,
	); err != nil {
		return StrapiProduct{}, fmt.Errorf("create Strapi product draft: %w", err)
	}
	if response.Data.DocumentID == "" {
		return StrapiProduct{}, fmt.Errorf("create Strapi product draft: response has no documentId")
	}

	return response.Data, nil
}

func (client *StrapiClient) FindProductByTelegramSourceID(
	ctx context.Context,
	telegramSourceID string,
) (*StrapiProduct, error) {
	telegramSourceID = strings.TrimSpace(telegramSourceID)
	if telegramSourceID == "" {
		return nil, fmt.Errorf("Telegram source ID is empty")
	}

	query := url.Values{}
	query.Set("status", "draft")
	query.Set("filters[telegramSourceId][$eq]", telegramSourceID)
	query.Set("pagination[pageSize]", "1")

	var response struct {
		Data []StrapiProduct `json:"data"`
	}
	if err := client.doJSON(
		ctx,
		http.MethodGet,
		"/api/products?"+query.Encode(),
		nil,
		&response,
	); err != nil {
		return nil, fmt.Errorf("find Strapi product by Telegram source ID: %w", err)
	}
	if len(response.Data) == 0 {
		return nil, nil
	}

	return &response.Data[0], nil
}

func (client *StrapiClient) PublishProduct(
	ctx context.Context,
	documentID string,
) (StrapiProduct, error) {
	if strings.TrimSpace(documentID) == "" {
		return StrapiProduct{}, fmt.Errorf("Strapi product documentId is empty")
	}

	body := struct {
		Data map[string]any `json:"data"`
	}{Data: map[string]any{}}

	var response struct {
		Data StrapiProduct `json:"data"`
	}
	path := "/api/products/" + url.PathEscape(documentID)
	if err := client.doJSON(ctx, http.MethodPut, path, body, &response); err != nil {
		return StrapiProduct{}, fmt.Errorf("publish Strapi product: %w", err)
	}

	return response.Data, nil
}

func (client *StrapiClient) doJSON(
	ctx context.Context,
	method string,
	path string,
	body any,
	destination any,
) error {
	var requestBody io.Reader
	if body != nil {
		bodyJSON, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		requestBody = bytes.NewReader(bodyJSON)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		method,
		client.baseURL+path,
		requestBody,
	)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+client.apiToken)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		errorBody, readErr := io.ReadAll(io.LimitReader(response.Body, 16*1024))
		if readErr != nil {
			return fmt.Errorf("Strapi returned %s; read error body: %w", response.Status, readErr)
		}
		return fmt.Errorf("Strapi returned %s: %s", response.Status, strings.TrimSpace(string(errorBody)))
	}

	if response.StatusCode == http.StatusNoContent {
		return nil
	}
	if destination == nil {
		if _, err := io.Copy(io.Discard, response.Body); err != nil {
			return fmt.Errorf("discard response body: %w", err)
		}
		return nil
	}
	if err := json.NewDecoder(response.Body).Decode(destination); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}

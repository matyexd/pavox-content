package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Notification failures must never retry or abort a product import.
type PublicationNotifier struct {
	client  *TelegramClient
	chatID  string
	siteURL string
}

func (client *TelegramClient) sendMessage(ctx context.Context, chatID, text string) error {
	values := url.Values{"chat_id": {chatID}, "text": {text}, "parse_mode": {"HTML"}, "disable_web_page_preview": {"true"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, telegramAPIBaseURL+"/bot"+client.token+"/sendMessage", strings.NewReader(values.Encode()))
	if err != nil {
		return fmt.Errorf("build Telegram notification")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return fmt.Errorf("send Telegram notification failed")
	}
	defer response.Body.Close()
	var result TelegramResponse[json.RawMessage]
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode Telegram notification response")
	}
	if response.StatusCode != http.StatusOK || !result.OK {
		return fmt.Errorf("Telegram notification rejected (code %d)", result.ErrorCode)
	}
	return nil
}

func (n *PublicationNotifier) send(ctx context.Context, post TelegramPost, status, detail string) {
	if n == nil || n.chatID == "" {
		return
	}
	text := fmt.Sprintf("<b>%s</b>\nПост: <code>%v</code>\nФото: %d", html.EscapeString(status), post.UpdateIDs, len(post.Photos))
	if detail != "" {
		text += "\n\n" + detail
	}
	if err := n.client.sendMessage(ctx, n.chatID, text); err != nil {
		fmt.Println("Publication notification:", err)
	}
}

func (n *PublicationNotifier) published(ctx context.Context, post TelegramPost, product StrapiProduct) {
	if n == nil {
		return
	}
	detail := "Товар опубликован."
	if base, err := url.Parse(n.siteURL); err == nil && base.Host != "" && (base.Scheme == "https" || base.Scheme == "http") && product.Slug != "" {
		link := strings.TrimRight(n.siteURL, "/") + "/product/" + url.PathEscape(product.Slug)
		detail += "\n<a href=\"" + html.EscapeString(link) + "\">Открыть на сайте ↗</a>"
	}
	n.send(ctx, post, "✅ Успешно опубликовано", detail)
}

// Keep diagnostics useful without exposing configured credentials to a chat.
func notificationError(err error) string {
	text := err.Error()
	for _, key := range []string{"TELEGRAM_BOT_TOKEN", "OPENAI_API_KEY", "STRAPI_API_TOKEN"} {
		if secret := os.Getenv(key); secret != "" {
			for _, value := range []string{secret, url.QueryEscape(secret), url.PathEscape(secret)} {
				text = strings.ReplaceAll(text, value, "[REDACTED]")
			}
		}
	}
	const maxRunes = 1200
	runes := []rune(text)
	if len(runes) > maxRunes {
		text = string(runes[:maxRunes]) + "…"
	}
	return text
}

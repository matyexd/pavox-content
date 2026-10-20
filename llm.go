package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

const productExtractionInstructions = `Сначала определи, описывает ли публикация конкретный товар.

Публикация считается товаром, если из текста можно определить конкретный продукт: его название или однозначный тип и характеристики. Цена, вес, состав и способ применения могут отсутствовать — это не делает публикацию нетоварной.

Если публикация не описывает конкретный товар, например это новость, поздравление, общий совет, анонс, личная история или реклама без определимого продукта:
- верни null в поле name;
- верни null во всех остальных строковых и числовых полях;
- верни пустой массив в category_slugs.

Если публикация описывает конкретный товар, поле name должно содержать его название и не должно быть null.

Извлекай значения только из текста Telegram-публикации. Не используй общие знания о товаре и не додумывай отсутствующие сведения.

Правила заполнения полей:
- description: кратко опиши свойства именно основного товара, явно указанные автором;
- compound: перечисли только конкретные ингредиенты, которые автор явно назвал. Не включай расплывчатые упоминания вроде «масла», «активы» или «компоненты» и не дополняй состав своими знаниями;
- method_application: заполняй только тогда, когда автор явно описал способ или последовательность применения. Упоминание того, что средство наносится, тает или действует на волосах или коже, само по себе не является инструкцией по применению;
- price: верни цену товара в российских рублях целым числом, только если она явно указана;
- variants: массив всех явно предлагаемых фасовок ОСНОВНОГО товара. Каждая запись содержит weight_grams, volume_ml и свою price (рубли). Например «50 мл — 700 руб., 100 мл — 1200 руб.» → две фасовки: {weight_grams:null,volume_ml:50,price:700} и {weight_grams:null,volume_ml:100,price:1200}.
- weight_grams: только явно указанный вес, «г», «гр», «грамм», «g». volume_ml: только явно указанный объём, «мл», «миллилитров», «ml». Никогда не записывай мл в граммы и не пересчитывай вес в объём. Явные кг и литры переводи в граммы и миллилитры (0,1 л = 100 мл); результат должен быть целым положительным числом. Дробные граммы/мл не округляй: неизвестный размер верни null.
- верхние price, weight_grams, volume_ml заполняй только для единственной фасовки. Если фасовок несколько — эти три верхних поля null; все пары размер–цена сохраняй в variants.
- Не связывай несколько размеров с одной ценой, если автор не указал однозначно, к каким фасовкам она относится. Не вычисляй цену пропорционально объёму. Не извлекай размеры ингредиентов и дозировки применения как фасовку. Неизвестную цену или размер верни null. Если нет ни цены, ни фасовки — variants пустой.
- Для нетоварного поста variants всегда пустой массив.

Определи основной товар, который продаётся или описывается в публикации. Не извлекай сведения и не назначай категории по упоминаниям других товаров, сравнений или рекомендаций.

Для category_slugs выбери одну или несколько наиболее конкретных подходящих категорий только из следующего списка:
%s

Правила выбора категорий:
- возвращай только slug, без названия категории;
- выбирай наиболее конкретную категорию;
- если выбрана дочерняя категория, не добавляй родительскую: Go добавит её самостоятельно;
- не выбирай категорию только потому, что соответствующее слово упоминается в тексте. Категория должна относиться именно к основному товару публикации;
- если подходящей категории нет или тип товара невозможно определить, верни пустой массив;
- category_slugs всегда должен быть массивом, а не null.`

type productExtractionInput struct {
	Caption string `json:"caption"`
	Text    string `json:"text"`
}

type ProductExtractor interface {
	Extract(ctx context.Context, post TelegramPost) (ProductExtraction, error)
}

type OpenAIProductExtractor struct {
	client *openai.Client
}

func (extractor OpenAIProductExtractor) Extract(
	ctx context.Context,
	post TelegramPost,
) (ProductExtraction, error) {
	return requestLLMHandle(ctx, extractor.client, post)
}

func productExtractionDeveloperMessage() string {
	var categories strings.Builder
	for _, category := range productCategories {
		fmt.Fprintf(&categories, "- %s — %s", category.Slug, category.Name)
		if category.ParentSlug != "" {
			fmt.Fprintf(&categories, " (дочерняя категория %s)", category.ParentSlug)
		}
		categories.WriteByte('\n')
	}

	return fmt.Sprintf(productExtractionInstructions, strings.TrimSpace(categories.String()))
}

func productExtractionRequest(post TelegramPost) (responses.ResponseNewParams, error) {
	inputJSON, err := json.Marshal(productExtractionInput{
		Caption: post.Caption,
		Text:    post.Text,
	})
	if err != nil {
		return responses.ResponseNewParams{}, fmt.Errorf("marshal LLM input: %w", err)
	}

	format := responses.ResponseFormatTextConfigParamOfJSONSchema(
		"product_extraction",
		productExtractionSchema(),
	)
	format.OfJSONSchema.Strict = openai.Bool(true)

	return responses.ResponseNewParams{
		Model: openai.ChatModelGPT6Luna,
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: responses.ResponseInputParam{
				responses.ResponseInputItemParamOfMessage(
					productExtractionDeveloperMessage(),
					responses.EasyInputMessageRoleDeveloper,
				),
				responses.ResponseInputItemParamOfMessage(
					string(inputJSON),
					responses.EasyInputMessageRoleUser,
				),
			},
		},
		Text: responses.ResponseTextConfigParam{
			Format: format,
		},
		Reasoning: responses.ReasoningParam{
			Effort: responses.ReasoningEffortLow,
		},
		Store: openai.Bool(false),
	}, nil
}

func responseLLMHandle(llmStringJSON string) (ProductExtraction, error) {
	var extraction ProductExtraction
	if err := json.Unmarshal([]byte(llmStringJSON), &extraction); err != nil {
		return ProductExtraction{}, fmt.Errorf("decode product extraction: %w", err)
	}

	return extraction, nil
}

func requestLLMHandle(
	ctx context.Context,
	client *openai.Client,
	post TelegramPost,
) (ProductExtraction, error) {
	if client == nil {
		return ProductExtraction{}, fmt.Errorf("OpenAI client is nil")
	}

	request, err := productExtractionRequest(post)
	if err != nil {
		return ProductExtraction{}, err
	}

	response, err := client.Responses.New(ctx, request)
	if err != nil {
		return ProductExtraction{}, fmt.Errorf("create OpenAI response: %w", err)
	}

	switch response.Status {
	case responses.ResponseStatusCompleted:
		// Continue below and inspect the response content.
	case responses.ResponseStatusFailed:
		return ProductExtraction{}, fmt.Errorf(
			"OpenAI response failed (%s): %s",
			response.Error.Code,
			response.Error.Message,
		)
	case responses.ResponseStatusIncomplete:
		return ProductExtraction{}, fmt.Errorf(
			"OpenAI response incomplete: %s",
			response.IncompleteDetails.Reason,
		)
	default:
		return ProductExtraction{}, fmt.Errorf(
			"unexpected OpenAI response status: %s",
			response.Status,
		)
	}

	for _, output := range response.Output {
		for _, content := range output.Content {
			if content.Type == "refusal" {
				return ProductExtraction{}, fmt.Errorf(
					"OpenAI refused product extraction: %s",
					content.Refusal,
				)
			}
		}
	}

	outputText := response.OutputText()
	if strings.TrimSpace(outputText) == "" {
		return ProductExtraction{}, fmt.Errorf("OpenAI response contains no output text")
	}

	return responseLLMHandle(outputText)
}

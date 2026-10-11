package main

func productExtractionSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "Product name extracted from the source post.",
			},
			"description": map[string]any{
				"type":        []string{"string", "null"},
				"description": "Product description, or null when it is not present in the source.",
			},
			"compound": map[string]any{
				"type":        []string{"string", "null"},
				"description": "Product ingredients, or null when they are not present in the source.",
			},
			"method_application": map[string]any{
				"type":        []string{"string", "null"},
				"description": "Usage instructions, or null when they are not present in the source.",
			},
			"price": map[string]any{
				"type":        []string{"integer", "null"},
				"description": "Product price in Russian rubles, or null when it is not present in the source.",
			},
			"weight_grams": map[string]any{
				"type":        []string{"integer", "null"},
				"description": "Product weight in grams, or null when it is not present in the source.",
			},
			"category_slugs": map[string]any{
				"type":        "array",
				"description": "Category slugs selected from the allowed categories supplied in the prompt.",
				"items": map[string]any{
					"type": "string",
				},
			},
		},
		"required": []string{
			"name",
			"description",
			"compound",
			"method_application",
			"price",
			"weight_grams",
			"category_slugs",
		},
		"additionalProperties": false,
	}
}

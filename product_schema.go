package main

func productExtractionSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        []string{"string", "null"},
				"description": "Product name, or null when the publication does not describe a specific product.",
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
			"volume_ml": map[string]any{
				"type":        []string{"integer", "null"},
				"description": "Explicit volume in millilitres for a single size only; never convert ml to grams.",
			},
			"variants": map[string]any{
				"type":        "array",
				"description": "All explicitly offered sizes of the main product, each with its own price. Empty if neither price nor size is stated.",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"weight_grams": map[string]any{"type": []string{"integer", "null"}},
						"volume_ml":    map[string]any{"type": []string{"integer", "null"}},
						"price":        map[string]any{"type": []string{"integer", "null"}},
					},
					"required":             []string{"weight_grams", "volume_ml", "price"},
					"additionalProperties": false,
				},
			},
			"category_slugs": map[string]any{
				"type":        "array",
				"description": "Category slugs selected from the allowed categories supplied in the prompt.",
				"items": map[string]any{
					"type": "string",
					"enum": productCategorySlugs(),
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
			"volume_ml",
			"variants",
			"category_slugs",
		},
		"additionalProperties": false,
	}
}

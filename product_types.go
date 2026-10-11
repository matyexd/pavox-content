package main

type ProductExtraction struct {
	Name              *string  `json:"name"`
	Description       *string  `json:"description"`
	Compound          *string  `json:"compound"`
	MethodApplication *string  `json:"method_application"`
	Price             *int     `json:"price"`
	WeightGrams       *int     `json:"weight_grams"`
	CategorySlugs     []string `json:"category_slugs"`
}

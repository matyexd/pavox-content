package main

type ProductExtraction struct {
	Name              *string                    `json:"name"`
	Description       *string                    `json:"description"`
	Compound          *string                    `json:"compound"`
	MethodApplication *string                    `json:"method_application"`
	Price             *int                       `json:"price"`
	WeightGrams       *int                       `json:"weight_grams"`
	VolumeMl          *int                       `json:"volume_ml"`
	Variants          []ProductVariantExtraction `json:"variants"`
	CategorySlugs     []string                   `json:"category_slugs"`
}

type ProductCategory struct {
	Name       string
	Slug       string
	ParentSlug string
}

var productCategories = []ProductCategory{
	{Name: "Для волос", Slug: "category"},
	{Name: "Для лица и тела", Slug: "category-1"},
	{Name: "Шампуни", Slug: "category-2", ParentSlug: "category"},
	{Name: "Мужская косметика", Slug: "category-3"},
	{Name: "ЭКО-бытовая химия", Slug: "category-4"},
	{Name: "SOS-уход", Slug: "category-5"},
	{Name: "Бальзамы и маски для волос", Slug: "category-6", ParentSlug: "category"},
	{Name: "Несмываемый уход за волосами", Slug: "category-7", ParentSlug: "category"},
}

func productCategorySlugs() []string {
	slugs := make([]string, 0, len(productCategories))
	for _, category := range productCategories {
		slugs = append(slugs, category.Slug)
	}

	return slugs
}

// Each price belongs to its explicit size; grams and millilitres are never interchangeable.
type ProductVariantExtraction struct {
	WeightGrams *int `json:"weight_grams"`
	VolumeMl    *int `json:"volume_ml"`
	Price       *int `json:"price"`
}

func (product ProductExtraction) ImportVariants() []ProductVariantExtraction {
	if len(product.Variants) > 0 {
		return product.Variants
	}
	if product.Price == nil && product.WeightGrams == nil && product.VolumeMl == nil {
		return []ProductVariantExtraction{}
	}
	return []ProductVariantExtraction{{WeightGrams: product.WeightGrams, VolumeMl: product.VolumeMl, Price: product.Price}}
}

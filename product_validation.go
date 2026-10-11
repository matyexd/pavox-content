package main

import (
	"fmt"
	"strings"
)

func validateProductExtraction(product ProductExtraction) error {
	if product.Name != nil && strings.TrimSpace(*product.Name) == "" {
		return fmt.Errorf("product name is empty")
	}

	allowedCategorySlugs := make(map[string]struct{}, len(productCategories))
	for _, category := range productCategories {
		allowedCategorySlugs[category.Slug] = struct{}{}
	}

	for _, slug := range product.CategorySlugs {
		if _, allowed := allowedCategorySlugs[slug]; !allowed {
			return fmt.Errorf("unknown product category slug: %q", slug)
		}
	}

	return nil
}

func addParentCategorySlugs(slugs []string) []string {
	categoriesBySlug := make(map[string]ProductCategory, len(productCategories))
	for _, category := range productCategories {
		categoriesBySlug[category.Slug] = category
	}

	result := make([]string, 0, len(slugs))
	seen := make(map[string]struct{}, len(slugs))

	appendSlug := func(slug string) {
		if _, exists := seen[slug]; exists {
			return
		}

		result = append(result, slug)
		seen[slug] = struct{}{}
	}

	for _, slug := range slugs {
		appendSlug(slug)

		category, exists := categoriesBySlug[slug]
		if exists && category.ParentSlug != "" {
			appendSlug(category.ParentSlug)
		}
	}

	return result
}

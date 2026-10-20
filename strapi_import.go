package main

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

var russianTransliteration = map[rune]string{
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "yo",
	'ж': "zh", 'з': "z", 'и': "i", 'й': "y", 'к': "k", 'л': "l", 'м': "m",
	'н': "n", 'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u",
	'ф': "f", 'х': "kh", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "shch",
	'ъ': "", 'ы': "y", 'ь': "", 'э': "e", 'ю': "yu", 'я': "ya",
}

func (client *StrapiClient) ImportProduct(
	ctx context.Context,
	post TelegramPost,
	product ProductExtraction,
	imagePaths []string,
) (StrapiProduct, bool, error) {
	if product.Name == nil {
		return StrapiProduct{}, false, fmt.Errorf("cannot import publication without a product")
	}

	sourceID, err := telegramSourceID(post)
	if err != nil {
		return StrapiProduct{}, false, err
	}

	existing, err := client.FindProductByTelegramSourceID(ctx, sourceID)
	if err != nil {
		return StrapiProduct{}, false, err
	}
	if err := validateProductExtraction(product); err != nil {
		return StrapiProduct{}, false, err
	}
	// Repeated imports update SKU prices without re-uploading images or replacing editorial content.
	if existing != nil {
		return client.ImportCatalogProduct(ctx, strapiProductInput(post, product, sourceID, nil, nil))
	}
	categoryIDs, err := client.CategoryIDsBySlugs(ctx, product.CategorySlugs)
	if err != nil {
		return StrapiProduct{}, false, err
	}
	mediaFiles, err := client.UploadFiles(ctx, imagePaths)
	if err != nil {
		return StrapiProduct{}, false, err
	}
	// An HTTP failure can follow a successful commit. Never delete media here: it may already be attached.
	return client.ImportCatalogProduct(ctx, strapiProductInput(post, product, sourceID, categoryIDs, mediaFiles))
}

func strapiProductInput(
	post TelegramPost,
	product ProductExtraction,
	sourceID string,
	categoryIDs []int,
	mediaFiles []StrapiMedia,
) StrapiProductInput {
	mediaIDs := make([]int, 0, len(mediaFiles))
	for _, media := range mediaFiles {
		mediaIDs = append(mediaIDs, media.ID)
	}

	var price *StrapiPrice
	if product.Price != nil {
		price = &StrapiPrice{
			Price:        *product.Price,
			CurrentPrice: *product.Price,
			Currency:     "RUB",
		}
	}

	variants := make([]StrapiVariantInput, 0)
	for _, variant := range product.ImportVariants() {
		variants = append(variants, StrapiVariantInput{WeightGrams: variant.WeightGrams, VolumeMl: variant.VolumeMl, Price: variant.Price})
	}
	name := strings.TrimSpace(*product.Name)
	return StrapiProductInput{
		Name:              name,
		Slug:              productSlug(name, post.UpdateIDs),
		TelegramSourceID:  sourceID,
		Description:       product.Description,
		Compound:          product.Compound,
		MethodApplication: product.MethodApplication,
		Price:             price,
		WeightGrams:       product.WeightGrams,
		VolumeMl:          product.VolumeMl,
		Variants:          variants,
		Categories:        categoryIDs,
		Media:             mediaIDs,
	}
}

func productSlug(name string, updateIDs []int) string {
	var slug strings.Builder
	lastWasSeparator := true

	for _, character := range strings.ToLower(strings.TrimSpace(name)) {
		part, isRussian := transliterateRussian(character)
		if isRussian {
			if part != "" {
				slug.WriteString(part)
				lastWasSeparator = false
			}
			continue
		}

		if character >= 'a' && character <= 'z' || character >= '0' && character <= '9' {
			slug.WriteRune(character)
			lastWasSeparator = false
			continue
		}

		if !lastWasSeparator {
			slug.WriteByte('-')
			lastWasSeparator = true
		}
	}

	base := strings.Trim(slug.String(), "-")
	if base == "" {
		base = "product"
	}
	if len(base) > 180 {
		base = strings.TrimRight(base[:180], "-")
	}

	ids := append([]int(nil), updateIDs...)
	sort.Ints(ids)
	idParts := make([]string, 0, len(ids))
	for index, id := range ids {
		if index > 0 && id == ids[index-1] {
			continue
		}
		idParts = append(idParts, strconv.Itoa(id))
	}
	if len(idParts) == 0 {
		return base
	}

	return base + "-" + strings.Join(idParts, "-")
}

func transliterateRussian(character rune) (string, bool) {
	part, exists := russianTransliteration[character]
	return part, exists
}

package main

import "testing"

func TestExtractedVariantUnits(t *testing.T) {
	cases := []struct {
		json             string
		grams, ml, price int
	}{
		{`{"name":"Крем","variants":[{"weight_grams":null,"volume_ml":50,"price":700}]}`, 0, 50, 700},
		{`{"name":"Скраб","variants":[{"weight_grams":300,"volume_ml":null,"price":1000}]}`, 300, 0, 1000},
		{`{"name":"Шампунь","volume_ml":250,"price":1200}`, 0, 250, 1200},
	}
	for _, tc := range cases {
		product, err := responseLLMHandle(tc.json)
		if err != nil {
			t.Fatal(err)
		}
		if err := validateProductExtraction(product); err != nil {
			t.Fatal(err)
		}
		variants := product.ImportVariants()
		if len(variants) != 1 {
			t.Fatal(variants)
		}
		v := variants[0]
		value := func(p *int) int {
			if p == nil {
				return 0
			}
			return *p
		}
		if value(v.WeightGrams) != tc.grams || value(v.VolumeMl) != tc.ml || value(v.Price) != tc.price {
			t.Fatalf("wrong units: %#v", v)
		}
	}
}

func TestInvalidVariantExtraction(t *testing.T) {
	for _, raw := range []string{
		`{"variants":[{"volume_ml":0,"price":100}]}`,
		`{"variants":[{"weight_grams":-1,"price":100}]}`,
		`{"variants":[{"volume_ml":50,"price":-100}]}`,
		`{"variants":[{"volume_ml":50,"price":100},{"volume_ml":50,"price":200}]}`,
	} {
		p, err := responseLLMHandle(raw)
		if err != nil {
			t.Fatal(err)
		}
		if validateProductExtraction(p) == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
}

func TestSingleSizeResponseIncludesVariant(t *testing.T) {
	for _, raw := range []string{
		`{"name":"Шампунь","volume_ml":250,"price":1300}`,
		`{"name":"Шампунь","volume_ml":250,"price":1300,"variants":[]}`,
		`{"name":"Шампунь","variants":[{"weight_grams":null,"volume_ml":250,"price":1300}]}`,
	} {
		product, err := responseLLMHandle(raw)
		if err != nil {
			t.Fatal(err)
		}
		if len(product.Variants) != 1 {
			t.Fatalf("single size missing from saved extraction: %#v", product)
		}
		input := strapiProductInput(TelegramPost{UpdateIDs: []int{831507937}}, product, "telegram:831507937", nil, nil)
		if len(input.Variants) != 1 {
			t.Fatalf("single size missing from import: %#v", input)
		}
		variant := input.Variants[0]
		if variant.VolumeMl == nil || *variant.VolumeMl != 250 || variant.Price == nil || *variant.Price != 1300 || variant.WeightGrams != nil {
			t.Fatalf("wrong shampoo variant: %#v", variant)
		}
	}
}

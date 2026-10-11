package main

import (
	"slices"
	"strings"
	"testing"
)

func TestValidateProductExtraction(t *testing.T) {
	validName := "Твёрдый бальзам"
	emptyName := "   "

	tests := []struct {
		name         string
		product      ProductExtraction
		wantError    bool
		errorMessage string
	}{
		{
			name: "valid product",
			product: ProductExtraction{
				Name:          &validName,
				CategorySlugs: []string{"category-6"},
			},
		},
		{
			name: "non-product publication",
			product: ProductExtraction{
				Name:          nil,
				CategorySlugs: []string{},
			},
		},
		{
			name: "empty product name",
			product: ProductExtraction{
				Name: &emptyName,
			},
			wantError:    true,
			errorMessage: "product name is empty",
		},
		{
			name: "unknown category",
			product: ProductExtraction{
				Name:          &validName,
				CategorySlugs: []string{"unknown-category"},
			},
			wantError:    true,
			errorMessage: `unknown product category slug: "unknown-category"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateProductExtraction(test.product)

			if !test.wantError {
				if err != nil {
					t.Fatalf("validateProductExtraction returned error: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatal("validateProductExtraction returned nil, want error")
			}
			if !strings.Contains(err.Error(), test.errorMessage) {
				t.Errorf("error = %q, want it to contain %q", err, test.errorMessage)
			}
		})
	}
}

func TestAddParentCategorySlugs(t *testing.T) {
	tests := []struct {
		name  string
		slugs []string
		want  []string
	}{
		{
			name:  "child category",
			slugs: []string{"category-6"},
			want:  []string{"category-6", "category"},
		},
		{
			name:  "top-level category",
			slugs: []string{"category-1"},
			want:  []string{"category-1"},
		},
		{
			name:  "children with the same parent",
			slugs: []string{"category-6", "category-2"},
			want:  []string{"category-6", "category", "category-2"},
		},
		{
			name:  "duplicate categories",
			slugs: []string{"category-6", "category", "category-6"},
			want:  []string{"category-6", "category"},
		},
		{
			name:  "no categories",
			slugs: nil,
			want:  []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := addParentCategorySlugs(test.slugs)
			if !slices.Equal(got, test.want) {
				t.Errorf("addParentCategorySlugs() = %v, want %v", got, test.want)
			}
		})
	}
}

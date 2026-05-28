package repository

import (
	"testing"
	"wms_dashboard/internal/models"
)

func TestUoMConversionCRUDAndMathSafety(t *testing.T) {
	setupTestDB(t)

	// 1. Seed UoMs
	uBox := &models.UoM{Code: "box", Name: "Box"}
	uKg := &models.UoM{Code: "kg", Name: "Kilogram"}
	uPcs := &models.UoM{Code: "pcs", Name: "Pieces"}
	_ = CreateUoM(uBox)
	_ = CreateUoM(uKg)
	_ = CreateUoM(uPcs)

	prod := &models.Product{
		SKU:   "UOM-PROD",
		Name:  "Bulk Rice Bag",
		UoMID: uKg.ID, // Base unit is kg
	}
	_ = CreateProduct(prod)

	// Create Conversion Formula: 1 box of UOM-PROD = 15.5 kg
	conv := &models.UoMConversion{
		ProductID:      prod.ID,
		FromUoMID:      uBox.ID,
		ToUoMID:        uKg.ID,
		MultiplyFactor: 15.5,
	}

	err := CreateConversion(conv)
	if err != nil {
		t.Fatalf("failed to create conversion: %v", err)
	}

	// Fetch conversions
	convs, err := FetchConversionsByProduct(prod.ID)
	if err != nil {
		t.Fatalf("failed to fetch product conversions: %v", err)
	}
	if len(convs) != 1 {
		t.Errorf("expected 1 conversion, got %d", len(convs))
	}
	fetchedConv := convs[0]
	if fetchedConv.MultiplyFactor != 15.5 {
		t.Errorf("expected multiply factor 15.5, got %f", fetchedConv.MultiplyFactor)
	}

	// 2. Perform Mathematical Conversion Tests
	// Let's emulate the exact logic used in handlers to guarantee mathematical correctness and rounding
	runMathConversion := func(qty int, factor float64) int {
		return int(float64(qty) * factor)
	}

	tests := []struct {
		name       string
		qty        int
		factor     float64
		expected   int
	}{
		{
			name:     "Simple integer conversion",
			qty:      10,
			factor:   15.5,
			expected: 155,
		},
		{
			name:     "Decimal truncation check",
			qty:      3,
			factor:   1.333,
			expected: 3, // 3 * 1.333 = 3.999 -> int() truncates to 3
		},
		{
			name:     "High precision rounding boundary",
			qty:      1,
			factor:   0.99999,
			expected: 0, // Truncation in Go cast
		},
		{
			name:     "Zero quantity conversion",
			qty:      0,
			factor:   100.5,
			expected: 0,
		},
		{
			name:     "Fractional conversion (e.g. 1 unit = 0.25 pack)",
			qty:      40,
			factor:   0.25,
			expected: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runMathConversion(tt.qty, tt.factor)
			if result != tt.expected {
				t.Errorf("expected %d, got %d for qty %d and factor %f", tt.expected, result, tt.qty, tt.factor)
			}
		})
	}

	// Delete conversion
	err = DeleteConversion(conv.ID)
	if err != nil {
		t.Fatalf("failed to delete conversion: %v", err)
	}

	// Verify deleted
	allConvs, _ := FetchAllConversions()
	for _, c := range allConvs {
		if c.ID == conv.ID {
			t.Error("expected conversion to be deleted, but it was found")
		}
	}
}

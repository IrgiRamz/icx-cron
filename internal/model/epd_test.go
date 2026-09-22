package model

import (
	"testing"
)

func TestCalculateEPD(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected int
	}{
		{
			name:     "Easycron Daily Job 22:00 (00 22 * * * *)",
			expr:     "00 22 * * * *",
			expected: 1,
		},
		{
			name:     "Easycron Daily Job 05:00 (00 5 * * * *)",
			expr:     "00 5 * * * *",
			expected: 1,
		},
		{
			name:     "Standard 5-field Daily Job 22:00 (0 22 * * *)",
			expr:     "0 22 * * *",
			expected: 1,
		},
		{
			name:     "Standard Every 10 Minutes (*/10 * * * *)",
			expr:     "*/10 * * * *",
			expected: 144,
		},
		{
			name:     "Easycron Multi-Hour Interval (0,10,20,30,40,50 6,7,8,9,10,11,12,13,14,15,16,17,18,19,20 * * * *)",
			expr:     "0,10,20,30,40,50 6,7,8,9,10,11,12,13,14,15,16,17,18,19,20 * * * *",
			expected: 90, // 6 times/hr * 15 hrs = 90
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateEPD(tt.expr)
			if got != tt.expected {
				t.Errorf("CalculateEPD(%q) = %d; want %d", tt.expr, got, tt.expected)
			}
		})
	}
}

package domain

import (
	"testing"
)

func TestEmojiForHealthFactor(t *testing.T) {
	tests := []struct {
		value    float64
		expected string
	}{
		{1.50, "🟩"},
		{1.49, "🟨"},
		{1.10, "🟨"},
		{1.09, "🟥"},
		{2.00, "🟩"},
		{0.95, "🟥"},
	}
	
	for _, test := range tests {
		result := EmojiForHealthFactor(test.value)
		if result != test.expected {
			t.Errorf("EmojiForHealthFactor(%.2f) = %s, expected %s", test.value, result, test.expected)
		}
	}
}

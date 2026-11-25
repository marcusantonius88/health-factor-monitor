package domain

import "math"

// EmojiForHealthFactor returns the visual emoji indicator for a health factor value.
// Rules:
//
//	HF >= 1.50 → 🟩 (safe)
//	1.10 <= HF < 1.50 → 🟨 (attention)
//	HF < 1.10 → 🟥 (critical)
func EmojiForHealthFactor(value float64) string {
	displayValue := math.Round(value*100) / 100
	if displayValue >= 1.50 {
		return "🟩"
	}
	if displayValue >= 1.10 {
		return "🟨"
	}
	return "🟥"
}

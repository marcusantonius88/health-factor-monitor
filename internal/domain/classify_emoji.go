package domain

// EmojiForHealthFactor returns the visual emoji indicator for a health factor value.
// Rules:
//   HF >= 1.50 → 🟩 (safe)
//   1.10 <= HF < 1.50 → 🟨 (attention)
//   HF < 1.10 → 🟥 (critical)
func EmojiForHealthFactor(value float64) string {
	if value >= 1.50 {
		return "🟩"
	}
	if value >= 1.10 {
		return "🟨"
	}
	return "🟥"
}

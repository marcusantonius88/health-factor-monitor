package cli

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/marcus/health-factor-monitor/internal/domain"
)

// positionLabel returns a human-readable position name, preferring the alias
// when set and falling back to the wallet address.
func positionLabel(p domain.LendingPosition) string {
	if strings.TrimSpace(p.Wallet.Alias) != "" {
		return p.Wallet.Alias
	}
	return p.Wallet.Address
}

// formatValue renders a success or error result for the Health Factor column.
func formatValue(r domain.ProviderResult) string {
	if r.HealthFactor != nil {
		return fmt.Sprintf("%.2f", r.HealthFactor.Value)
	}
	return "-"
}

// formatStatus renders the classification label, or an error marker when the
// provider failed.
func formatStatus(r domain.ProviderResult) string {
	if r.HealthFactor != nil {
		return r.HealthFactor.Classification
	}
	return "ERROR"
}

// FormatResults renders an aligned table of provider results. On error it also
// appends the underlying error message beneath the table.
func FormatResults(results []domain.ProviderResult) string {
	var buf strings.Builder
	w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)

	fmt.Fprintln(w, "Position\tProtocol\tNetwork\tHealth Factor\tStatus")
	for _, r := range results {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			positionLabel(r.Position),
			r.Position.Protocol,
			r.Position.Network,
			formatValue(r),
			formatStatus(r),
		)
	}
	_ = w.Flush()

	for _, r := range results {
		if r.Error != "" {
			fmt.Fprintf(&buf, "\nError [%s]: %s\n", positionLabel(r.Position), r.Error)
		}
	}

	return buf.String()
}

// HasSuccess reports whether at least one health factor was successfully
// retrieved.
func HasSuccess(results []domain.ProviderResult) bool {
	for _, r := range results {
		if r.HealthFactor != nil {
			return true
		}
	}
	return false
}

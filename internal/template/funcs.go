package template

import (
	"fmt"
	"time"

	"github.com/museigen/lore/internal/domain"
)

// formatDate accepts both time.Time and string (already formatted YYYY-MM-DD).
// This allows usage with TemplateContext.Date (string) and external time.Time values.
func formatDate(v any) string {
	switch d := v.(type) {
	case time.Time:
		if d.IsZero() {
			return ""
		}
		return d.Format("2006-01-02")
	case string:
		return d
	default:
		return fmt.Sprintf("%v", v)
	}
}

// CONSOLIDATE: envisager extraction vers domain/ si les import boundaries le permettent
func slugify(s string) string {
	return domain.Slugify(s)
}

func commitLink(ref string) string {
	if ref == "" {
		return ""
	}
	return fmt.Sprintf("[`%.7s`](../../commit/%s)", ref, ref)
}

package template

import (
	"time"

	"github.com/museigen/lore/internal/domain"
)

func formatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

func slugify(s string) string {
	return domain.Slugify(s)
}

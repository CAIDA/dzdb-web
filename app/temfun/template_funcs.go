package temfun

import (
	"html/template"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// Funcs is a global FuncMap for all tempaltes to include
var Funcs = make(template.FuncMap)
var fmt = message.NewPrinter(language.English)

func init() {
	Funcs["count"] = count
	Funcs["nfmt"] = nfmt
	Funcs["date"] = date
}

func count(count int, totalCount *int64) string {
	if totalCount == nil || int64(count) == *totalCount {
		return fmt.Sprintf("%d", count)
	}
	return fmt.Sprintf("%d of %d", count, *totalCount)
}

func nfmt(count int64) string {
	return fmt.Sprintf("%d", count)
}

func date(date *time.Time) string {
	if date == nil {
		return ""
	}
	return date.Format("Jan 02, 2006")
}

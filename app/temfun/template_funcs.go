// Package temfun defines functions included in the html templates
package temfun

import (
	"html/template"
	"strings"
	"time"

	"golang.org/x/net/idna"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// Funcs is a global FuncMap for all templates to include
var Funcs = make(template.FuncMap)
var fmt = message.NewPrinter(language.English)

func init() {
	Funcs["count"] = count
	Funcs["nfmt"] = nfmt
	Funcs["date"] = date
	Funcs["drefInt"] = defrefInt
	Funcs["toUnicode"] = toUnicode
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

func defrefInt(i *int64) int64 {
	return *i
}

func toUnicode(s string) string {
	lcStr := strings.ToLower(s)
	uniStr, _ := idna.ToUnicode(lcStr)
	if uniStr == lcStr {
		return lcStr
	} else {
		return fmt.Sprintf("%v (%v)", lcStr, uniStr)
	}
}

package mongodb

import "regexp"

var validQueryValue = regexp.MustCompile(`^[A-Za-z0-9:._@+\-]{1,512}$`)

func sanitizeValue(val string) string {
	if !validQueryValue.MatchString(val) {
		return ""
	}
	return val
}

func sanitizeRegex(val string) string {
	return regexp.QuoteMeta(val)
}

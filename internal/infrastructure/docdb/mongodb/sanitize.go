package mongodb

import "regexp"

var validQueryValue = regexp.MustCompile(`^[^\x00]*$`)

func sanitizeValue(val string) string {
	if !validQueryValue.MatchString(val) {
		return ""
	}
	return val
}

func sanitizeRegex(val string) string {
	return regexp.QuoteMeta(val)
}

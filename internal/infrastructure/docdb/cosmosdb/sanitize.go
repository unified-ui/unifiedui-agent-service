// Package cosmosdb provides input sanitization for CosmosDB.
package cosmosdb

import "regexp"

var validQueryValue = regexp.MustCompile(`^[A-Za-z0-9:._@+\-]{1,512}$`)

func sanitizeValue(val string) string {
	if !validQueryValue.MatchString(val) {
		return ""
	}
	return val
}

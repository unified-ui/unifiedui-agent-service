package foundry

import "regexp"

var validIdentifier = regexp.MustCompile(`^[A-Za-z0-9_\-.:]{1,512}$`)

func validateIdentifier(val string) string {
	if !validIdentifier.MatchString(val) {
		return ""
	}
	return val
}

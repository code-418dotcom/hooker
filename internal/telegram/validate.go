package telegram

import "regexp"

// validName matches Docker container and group names: alphanumeric, dash, underscore, dot.
var validName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// isValidName returns true if the name is a valid container or group identifier.
func isValidName(name string) bool {
	return len(name) > 0 && len(name) <= 128 && validName.MatchString(name)
}

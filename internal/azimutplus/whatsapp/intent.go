package whatsapp

import "strings"

func ParseYesIntent(text string) bool {
	normalized := strings.TrimSpace(strings.ToLower(text))
	switch normalized {
	case "oui", "yes", "y", "ok", "ok!", "go", "dispo":
		return true
	default:
		return false
	}
}

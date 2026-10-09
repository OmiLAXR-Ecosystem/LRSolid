package xapi

import "strings"

const (
	Version103 = "1.0.3"
	Version200 = "2.0.0"

	HeaderVersion           = "X-Experience-API-Version"
	HeaderConsistentThrough = "X-Experience-API-Consistent-Through"
	HeaderHash              = "X-Experience-API-Hash"

	VerbVoided          = "http://adlnet.gov/expapi/verbs/voided"
	AttachmentSignature = "http://adlnet.gov/expapi/attachments/signature"
)

// Negotiate maps a client X-Experience-API-Version value to the highest
// supported patch version of that major.minor. Empty or unknown values fail.
func Negotiate(header string) (string, bool) {
	header = strings.TrimSpace(header)
	if header == "" {
		return "", false
	}
	parts := strings.Split(header, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return "", false
	}
	for _, p := range parts {
		if p == "" || !digits(p) {
			return "", false
		}
	}
	switch parts[0] + "." + parts[1] {
	case "1.0":
		return Version103, true
	case "2.0":
		return Version200, true
	default:
		return "", false
	}
}

func Major(version string) string {
	if strings.HasPrefix(version, "2.") {
		return "2"
	}
	return "1"
}

func digits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

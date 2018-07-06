package backplane

import "regexp"

// Authorization lists certificate names that may access the backplane
type Authorization struct {
	Insecure bool     `json:"insecure" yaml:"insecure"`
	Full     []string `json:"full" yaml:"full"`
	RO       []string `json:"read_only" yaml:"read_only"`
}

// ROAllowed determines if this user can access read only action
func (a *Authorization) ROAllowed(c string) bool {
	if a.Insecure {
		return true
	}

	if a.FullAllowed(c) {
		return true
	}

	if matchAnyRegex([]byte(c), a.RO) {
		return true
	}

	return false
}

// FullAllowed determines if this user can access all actions
func (a *Authorization) FullAllowed(c string) bool {
	if a.Insecure {
		return true
	}

	if matchAnyRegex([]byte(c), a.Full) {
		return true
	}

	return false
}

func matchAnyRegex(str []byte, regex []string) bool {
	for _, reg := range regex {
		if matched, _ := regexp.Match(reg, str); matched {
			return true
		}
	}

	return false
}

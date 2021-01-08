// endpoints_test.go
package main

import (
	"regexp"
	"testing"
)

func TestInvalidID(t *testing.T) {
	// Pass non compliant xname to test

	xname := "test&123"

	invalid := invalidID(xname)

	if !invalid {
		t.Errorf("invalid xname returned as compliant: %s",
			xname)
	}
}

// pulled from api_token.go
func invalidID(id string) bool {

	var validID = regexp.MustCompile(`^[a-zA-Z0-9]*$`)

	if id == "" {
		return true
	}
	return !validID.MatchString(id)
}

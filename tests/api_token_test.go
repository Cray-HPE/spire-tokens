// endpoints_test.go
package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tokens "stash.us.cray.com/spet/spire-tokens/go"
)

func TestGenerateToken(t *testing.T) {
	// Pass compliant xname to test
	xname := strings.NewReader(`xname=test123`)
	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	sendRequest(xname, rr, t)

	// Expecting error as there is no available socket
	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.
	expected := "Error while dialing dial unix /tmp/spire-registration.sock"

	if !strings.ContainsAny(rr.Body.String(), expected) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

func TestGenerateTokenBadXname(t *testing.T) {
	// Pass compliant xname to test
	xname := strings.NewReader(`xname=&123`)
	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	sendRequest(xname, rr, t)

	// Expecting error bc there is no available socket
	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.
	expected := "Invalid or no xname provided"

	if !strings.ContainsAny(rr.Body.String(), expected) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

func sendRequest(xname *strings.Reader, rr *httptest.ResponseRecorder, t *testing.T) {
	req, err := http.NewRequest("POST", "/api/token", xname)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	handler := http.HandlerFunc(tokens.GenerateToken)

	handler.ServeHTTP(rr, req)
}

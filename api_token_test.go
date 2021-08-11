// endpoints_test.go
package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	tokens "github.com/Cray-HPE/spire-tokens/go"
	"github.com/spiffe/spire/pkg/server/plugin/datastore"
	"github.com/spiffe/spire/test/fakes/fakedatastore"
	"github.com/spiffe/spire/test/fakes/fakeregistrationclient"
	"github.com/stretchr/testify/require"
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

func TestCreateToken(t *testing.T) {
	ds := fakedatastore.New(t)
	ctx := context.Background()
	c := fakeregistrationclient.New(t, "spiffe://shasta", ds, nil)
	ttl := 6000
	token, err := tokens.CreateToken(ctx, c, ttl)

	if err != nil {
		t.Errorf("Failed to create token : %v", err)
	}

	var validToken = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89aAbB][a-f0-9]{3}-[a-f0-9]{12}$`)

	if !validToken.MatchString(token) {
		t.Errorf("CreateToken did not return a valid UUID: %v", token)
	}
}

func TestCreateRegistrationRecord(t *testing.T) {
	ds := fakedatastore.New(t)
	ctx := context.Background()
	c := fakeregistrationclient.New(t, "spiffe://shasta", ds, nil)
	parentID := "spiffe://shasta/spire/agent/join_token/TOKEN"
	spiffeID := "spiffe://shasta/ncn/xname"
	err := tokens.CreateRegistrationRecord(ctx, c, parentID, spiffeID)

	if err != nil {
		t.Errorf("Failed to create registration record: %v", err)
	}
	regEntries, err := ds.ListRegistrationEntries(ctx, &datastore.ListRegistrationEntriesRequest{})
	if err != nil {
		t.Errorf("Failed to request registration entries: %v", err)
	}
	require.Equal(t, regEntries.Entries[0].SpiffeId, "spiffe://shasta/ncn/xname")
	require.Equal(t, regEntries.Entries[0].ParentId, "spiffe://shasta/spire/agent/join_token/TOKEN")
	require.Equal(t, regEntries.Entries[0].Selectors[0].Value, "spiffe://shasta/spire/agent/join_token/TOKEN")
}

func TestParseWorkloads(t *testing.T) {
	actual, err := tokens.ParseWorkloads("tests/workloads_test.yaml")
	if err != nil {
		t.Errorf("Failed to parse workload test file: %v", err)
	}

	var expected = []tokens.Workload{
		{
			ParentID: "spiffe://shasta/ncn",
			SpiffeID: "spiffe://shasta/ncn/XNAME/workload1",
			Selectors: []tokens.WorkloadSelector{
				{Type: "unix", Value: "uid:0"},
				{Type: "unix", Value: "gid:0"},
			},
		},
		{
			ParentID: "spiffe://shasta/ncn",
			SpiffeID: "spiffe://shasta/ncn/XNAME/workload2",
			Selectors: []tokens.WorkloadSelector{
				{Type: "unix", Value: "uid:0"},
				{Type: "unix", Value: "gid:0"},
			},
			Ttl: 634000,
		},
	}
	require.Equal(t, expected, actual)
}
func TestCreateWorkloads(t *testing.T) {
	ds := fakedatastore.New(t)
	ctx := context.Background()
	c := fakeregistrationclient.New(t, "spiffe://shasta", ds, nil)
	xnames := []string{"xname1", "xname2"}
	var workloads = []tokens.Workload{
		{
			ParentID: "spiffe://shasta/ncn",
			SpiffeID: "spiffe://shasta/ncn/XNAME/workload1",
			Selectors: []tokens.WorkloadSelector{
				{Type: "unix", Value: "uid:0"},
				{Type: "unix", Value: "gid:0"},
			},
		},
		{
			ParentID: "spiffe://shasta/ncn",
			SpiffeID: "spiffe://shasta/ncn/XNAME/workload2",
			Selectors: []tokens.WorkloadSelector{
				{Type: "unix", Value: "uid:0"},
				{Type: "unix", Value: "gid:0"},
			},
			Ttl: 634000,
		},
	}

	for _, xname := range xnames {
		err := tokens.CreateWorkloads(ctx, c, xname, workloads)

		if err != nil {
			t.Errorf("Failed to create registration record: %v", err)
		}
	}

	regEntries, err := ds.ListRegistrationEntries(ctx, &datastore.ListRegistrationEntriesRequest{})
	if err != nil {
		t.Errorf("Failed to request registration entries: %v", err)
	}

	require.Equal(t, regEntries.Entries[0].SpiffeId, "spiffe://shasta/ncn/xname1/workload1")
	require.Equal(t, regEntries.Entries[0].ParentId, "spiffe://shasta/ncn")
	require.Equal(t, regEntries.Entries[0].Selectors[0].Value, "gid:0")
	require.Equal(t, regEntries.Entries[0].Selectors[1].Value, "uid:0")
	require.Equal(t, regEntries.Entries[1].SpiffeId, "spiffe://shasta/ncn/xname1/workload2")
	require.Equal(t, regEntries.Entries[1].ParentId, "spiffe://shasta/ncn")
	require.Equal(t, regEntries.Entries[1].Selectors[0].Value, "gid:0")
	require.Equal(t, regEntries.Entries[1].Selectors[1].Value, "uid:0")
	require.Equal(t, regEntries.Entries[1].Ttl, int32(634000))
	require.Equal(t, regEntries.Entries[2].SpiffeId, "spiffe://shasta/ncn/xname2/workload1")
	require.Equal(t, regEntries.Entries[2].ParentId, "spiffe://shasta/ncn")
	require.Equal(t, regEntries.Entries[2].Selectors[0].Value, "gid:0")
	require.Equal(t, regEntries.Entries[2].Selectors[1].Value, "uid:0")
	require.Equal(t, regEntries.Entries[3].SpiffeId, "spiffe://shasta/ncn/xname2/workload2")
	require.Equal(t, regEntries.Entries[3].ParentId, "spiffe://shasta/ncn")
	require.Equal(t, regEntries.Entries[3].Selectors[0].Value, "gid:0")
	require.Equal(t, regEntries.Entries[3].Selectors[1].Value, "uid:0")
	require.Equal(t, regEntries.Entries[3].Ttl, int32(634000))
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

/*
 *
 *  MIT License
 *
 *  (C) Copyright 2022 Hewlett Packard Enterprise Development LP
 *
 *  Permission is hereby granted, free of charge, to any person obtaining a
 *  copy of this software and associated documentation files (the "Software"),
 *  to deal in the Software without restriction, including without limitation
 *  the rights to use, copy, modify, merge, publish, distribute, sublicense,
 *  and/or sell copies of the Software, and to permit persons to whom the
 *  Software is furnished to do so, subject to the following conditions:
 *
 *  The above copyright notice and this permission notice shall be included
 *  in all copies or substantial portions of the Software.
 *
 *  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 *  IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 *  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL
 *  THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR
 *  OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,
 *  ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
 *  OTHER DEALINGS IN THE SOFTWARE.
 *
 */
// endpoints_test.go
package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	tokens "github.com/Cray-HPE/spire-tokens/go"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/spire/pkg/server/datastore"
	"github.com/spiffe/spire/test/fakes/fakedatastore"
	"github.com/spiffe/spire/test/fakes/fakeentryclient"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("SPIRE_DOMAIN", "shasta")
}

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
	expected := "Error while dialing dial unix /tmp/spire-server/private/api.sock"

	if !strings.ContainsAny(rr.Body.String(), expected) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

/* There doesn't appear to be a fake for entry agent
  func TestCreateToken(t *testing.T) {
	ds := fakedatastore.New(t)
	ctx := context.Background()

	td, err := spiffeid.TrustDomainFromString("spiffe://shasta")
	if err != nil {
		t.Errorf("Failed to create trust domain: %v", err)
	}

	c := fakeentryagent.New(t, td, ds, nil)

	ttl := 6000
	token, err := tokens.CreateToken(ctx, c, ttl)
	if err != nil {
		t.Errorf("Failed to create token : %v", err)
	}

	validToken := regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89aAbB][a-f0-9]{3}-[a-f0-9]{12}$`)

	if !validToken.MatchString(token) {
		t.Errorf("CreateToken did not return a valid UUID: %v", token)
	}
} */

func TestCreateRegistrationRecord(t *testing.T) {
	ds := fakedatastore.New(t)
	ctx := context.Background()
	td, err := spiffeid.TrustDomainFromString("spiffe://shasta")
	if err != nil {
		t.Errorf("Failed to create trust domain: %v", err)
	}

	c := fakeentryclient.New(t, td, ds, nil)
	parentID := "/spire/agent/join_token/TOKEN"
	spiffeID := "/ncn/xname"

	err = tokens.CreateRegistrationRecord(ctx, c, parentID, spiffeID)
	if err != nil {
		t.Errorf("Failed to create registration record: %v", err)
	}
	regEntries, err := ds.ListRegistrationEntries(ctx, &datastore.ListRegistrationEntriesRequest{})
	if err != nil {
		t.Errorf("Failed to request registration entries: %v", err)
	}
	require.Equal(t, "spiffe://shasta/ncn/xname", regEntries.Entries[0].SpiffeId)
	require.Equal(t, "spiffe://shasta/spire/agent/join_token/TOKEN", regEntries.Entries[0].ParentId)
	require.Equal(t, "spiffe://shasta/spire/agent/join_token/TOKEN", regEntries.Entries[0].Selectors[0].Value)
}

func TestParseWorkloads(t *testing.T) {
	actual, err := tokens.ParseWorkloads("tests/workloads_test.yaml")
	if err != nil {
		t.Errorf("Failed to parse workload test file: %v", err)
	}

	expected := []tokens.Workload{
		{
			SpiffeID: "/ncn/XNAME/workload1",
			Selectors: []tokens.WorkloadSelector{
				{Type: "unix", Value: "uid:0"},
				{Type: "unix", Value: "gid:0"},
			},
		},
		{
			SpiffeID: "/ncn/XNAME/workload2",
			Selectors: []tokens.WorkloadSelector{
				{Type: "unix", Value: "uid:0"},
				{Type: "unix", Value: "gid:0"},
			},
			Ttl: 634000,
		},
	}
	require.Equal(t, actual, expected)
}

func TestCreateWorkloadsCompute(t *testing.T) {
	ds := fakedatastore.New(t)
	ctx := context.Background()

	td, err := spiffeid.TrustDomainFromString("spiffe://shasta")
	if err != nil {
		t.Errorf("Failed to create trust domain: %v", err)
	}

	c := fakeentryclient.New(t, td, ds, nil)
	xnames := []string{"compute1", "compute2"}
	workloads := []tokens.Workload{
		{
			SpiffeID: "/test/XNAME/workload1",
			Selectors: []tokens.WorkloadSelector{
				{Type: "unix", Value: "uid:0"},
				{Type: "unix", Value: "gid:0"},
			},
		},
		{
			SpiffeID: "/test/XNAME/workload2",
			Selectors: []tokens.WorkloadSelector{
				{Type: "unix", Value: "uid:0"},
				{Type: "unix", Value: "gid:0"},
			},
			Ttl: 634000,
		},
	}

	for _, xname := range xnames {
		err := tokens.CreateWorkloads(ctx, c, xname, workloads, "compute")
		if err != nil {
			t.Errorf("Failed to create registration record: %v", err)
		}
	}

	regEntries, err := ds.ListRegistrationEntries(ctx, &datastore.ListRegistrationEntriesRequest{})
	if err != nil {
		t.Errorf("Failed to request registration entries: %v", err)
	}

	require.Equal(t, "spiffe://shasta/test/compute1/workload1", regEntries.Entries[0].SpiffeId)
	require.Equal(t, "spiffe://shasta/compute/tenant1/compute1", regEntries.Entries[0].ParentId)
	require.Equal(t, "gid:0", regEntries.Entries[0].Selectors[0].Value)
	require.Equal(t, "uid:0", regEntries.Entries[0].Selectors[1].Value)
	require.Equal(t, "spiffe://shasta/test/compute1/workload2", regEntries.Entries[1].SpiffeId)
	require.Equal(t, "spiffe://shasta/compute/tenant1/compute1", regEntries.Entries[1].ParentId)
	require.Equal(t, "gid:0", regEntries.Entries[1].Selectors[0].Value)
	require.Equal(t, "uid:0", regEntries.Entries[1].Selectors[1].Value)
	require.Equal(t, int32(634000), regEntries.Entries[1].Ttl)
	require.Equal(t, "spiffe://shasta/test/compute2/workload1", regEntries.Entries[2].SpiffeId)
	require.Equal(t, "spiffe://shasta/compute/tenant1/compute2", regEntries.Entries[2].ParentId)
	require.Equal(t, "gid:0", regEntries.Entries[2].Selectors[0].Value)
	require.Equal(t, "uid:0", regEntries.Entries[2].Selectors[1].Value)
	require.Equal(t, "spiffe://shasta/test/compute2/workload2", regEntries.Entries[3].SpiffeId)
	require.Equal(t, "spiffe://shasta/compute/tenant1/compute2", regEntries.Entries[3].ParentId)
	require.Equal(t, "gid:0", regEntries.Entries[3].Selectors[0].Value)
	require.Equal(t, "uid:0", regEntries.Entries[3].Selectors[1].Value)
	require.Equal(t, int32(634000), regEntries.Entries[3].Ttl)
}

func TestCreateWorkloadsStorage(t *testing.T) {
	ds := fakedatastore.New(t)
	ctx := context.Background()

	td, err := spiffeid.TrustDomainFromString("spiffe://shasta")
	if err != nil {
		t.Errorf("Failed to create trust domain: %v", err)
	}

	c := fakeentryclient.New(t, td, ds, nil)
	xnames := []string{"storage1", "storage2"}
	workloads := []tokens.Workload{
		{
			SpiffeID: "/test/XNAME/workload1",
			Selectors: []tokens.WorkloadSelector{
				{Type: "unix", Value: "uid:0"},
				{Type: "unix", Value: "gid:0"},
			},
		},
		{
			SpiffeID: "/test/XNAME/workload2",
			Selectors: []tokens.WorkloadSelector{
				{Type: "unix", Value: "uid:0"},
				{Type: "unix", Value: "gid:0"},
			},
			Ttl: 634000,
		},
	}

	for _, xname := range xnames {
		err := tokens.CreateWorkloads(ctx, c, xname, workloads, "storage")
		if err != nil {
			t.Errorf("Failed to create registration record: %v", err)
		}
	}

	regEntries, err := ds.ListRegistrationEntries(ctx, &datastore.ListRegistrationEntriesRequest{})
	if err != nil {
		t.Errorf("Failed to request registration entries: %v", err)
	}

	require.Equal(t, "spiffe://shasta/test/storage1/workload1", regEntries.Entries[0].SpiffeId)
	require.Equal(t, "spiffe://shasta/storage/tenant1/storage1", regEntries.Entries[0].ParentId)
	require.Equal(t, "gid:0", regEntries.Entries[0].Selectors[0].Value)
	require.Equal(t, "uid:0", regEntries.Entries[0].Selectors[1].Value)
	require.Equal(t, "spiffe://shasta/test/storage1/workload2", regEntries.Entries[1].SpiffeId)
	require.Equal(t, "spiffe://shasta/storage/tenant1/storage1", regEntries.Entries[1].ParentId)
	require.Equal(t, "gid:0", regEntries.Entries[1].Selectors[0].Value)
	require.Equal(t, "uid:0", regEntries.Entries[1].Selectors[1].Value)
	require.Equal(t, int32(634000), regEntries.Entries[1].Ttl)
	require.Equal(t, "spiffe://shasta/test/storage2/workload1", regEntries.Entries[2].SpiffeId)
	require.Equal(t, "spiffe://shasta/storage/tenant1/storage2", regEntries.Entries[2].ParentId)
	require.Equal(t, "gid:0", regEntries.Entries[2].Selectors[0].Value)
	require.Equal(t, "uid:0", regEntries.Entries[2].Selectors[1].Value)
	require.Equal(t, "spiffe://shasta/test/storage2/workload2", regEntries.Entries[3].SpiffeId)
	require.Equal(t, "spiffe://shasta/storage/tenant1/storage2", regEntries.Entries[3].ParentId)
	require.Equal(t, "gid:0", regEntries.Entries[3].Selectors[0].Value)
	require.Equal(t, "uid:0", regEntries.Entries[3].Selectors[1].Value)
	require.Equal(t, int32(634000), regEntries.Entries[3].Ttl)
}

func TestCreateWorkloadsNcn(t *testing.T) {
	ds := fakedatastore.New(t)
	ctx := context.Background()

	td, err := spiffeid.TrustDomainFromString("spiffe://shasta")
	if err != nil {
		t.Errorf("Failed to create trust domain: %v", err)
	}

	c := fakeentryclient.New(t, td, ds, nil)
	xnames := []string{"ncn1", "ncn2"}
	workloads := []tokens.Workload{
		{
			SpiffeID: "/test/XNAME/workload1",
			Selectors: []tokens.WorkloadSelector{
				{Type: "unix", Value: "uid:0"},
				{Type: "unix", Value: "gid:0"},
			},
		},
		{
			SpiffeID: "/test/XNAME/workload2",
			Selectors: []tokens.WorkloadSelector{
				{Type: "unix", Value: "uid:0"},
				{Type: "unix", Value: "gid:0"},
			},
			Ttl: 634000,
		},
	}

	for _, xname := range xnames {
		err := tokens.CreateWorkloads(ctx, c, xname, workloads, "ncn")
		if err != nil {
			t.Errorf("Failed to create registration record: %v", err)
		}
	}

	regEntries, err := ds.ListRegistrationEntries(ctx, &datastore.ListRegistrationEntriesRequest{})
	if err != nil {
		t.Errorf("Failed to request registration entries: %v", err)
	}

	require.Equal(t, "spiffe://shasta/test/ncn1/workload1", regEntries.Entries[0].SpiffeId)
	require.Equal(t, "spiffe://shasta/ncn/tenant1/ncn1", regEntries.Entries[0].ParentId)
	require.Equal(t, "gid:0", regEntries.Entries[0].Selectors[0].Value)
	require.Equal(t, "uid:0", regEntries.Entries[0].Selectors[1].Value)
	require.Equal(t, "spiffe://shasta/test/ncn1/workload2", regEntries.Entries[1].SpiffeId)
	require.Equal(t, "spiffe://shasta/ncn/tenant1/ncn1", regEntries.Entries[1].ParentId)
	require.Equal(t, "gid:0", regEntries.Entries[1].Selectors[0].Value)
	require.Equal(t, "uid:0", regEntries.Entries[1].Selectors[1].Value)
	require.Equal(t, int32(634000), regEntries.Entries[1].Ttl)
	require.Equal(t, "spiffe://shasta/test/ncn2/workload1", regEntries.Entries[2].SpiffeId)
	require.Equal(t, "spiffe://shasta/ncn/tenant1/ncn2", regEntries.Entries[2].ParentId)
	require.Equal(t, "gid:0", regEntries.Entries[2].Selectors[0].Value)
	require.Equal(t, "uid:0", regEntries.Entries[2].Selectors[1].Value)
	require.Equal(t, "spiffe://shasta/test/ncn2/workload2", regEntries.Entries[3].SpiffeId)
	require.Equal(t, "spiffe://shasta/ncn/tenant1/ncn2", regEntries.Entries[3].ParentId)
	require.Equal(t, "gid:0", regEntries.Entries[3].Selectors[0].Value)
	require.Equal(t, "uid:0", regEntries.Entries[3].Selectors[1].Value)
	require.Equal(t, int32(634000), regEntries.Entries[3].Ttl)
}

func TestCreateWorkloadsUAN(t *testing.T) {
	ds := fakedatastore.New(t)
	ctx := context.Background()

	td, err := spiffeid.TrustDomainFromString("spiffe://shasta")
	if err != nil {
		t.Errorf("Failed to create trust domain: %v", err)
	}

	c := fakeentryclient.New(t, td, ds, nil)
	xnames := []string{"uan1", "uan2"}
	workloads := []tokens.Workload{
		{
			SpiffeID: "/test/XNAME/workload1",
			Selectors: []tokens.WorkloadSelector{
				{Type: "unix", Value: "uid:0"},
				{Type: "unix", Value: "gid:0"},
			},
		},
		{
			SpiffeID: "/test/XNAME/workload2",
			Selectors: []tokens.WorkloadSelector{
				{Type: "unix", Value: "uid:0"},
				{Type: "unix", Value: "gid:0"},
			},
			Ttl: 634000,
		},
	}

	for _, xname := range xnames {
		err := tokens.CreateWorkloads(ctx, c, xname, workloads, "uan")
		if err != nil {
			t.Errorf("Failed to create registration record: %v", err)
		}
	}

	regEntries, err := ds.ListRegistrationEntries(ctx, &datastore.ListRegistrationEntriesRequest{})
	if err != nil {
		t.Errorf("Failed to request registration entries: %v", err)
	}

	require.Equal(t, "spiffe://shasta/test/uan1/workload1", regEntries.Entries[0].SpiffeId)
	require.Equal(t, "spiffe://shasta/uan/tenant1/uan1", regEntries.Entries[0].ParentId)
	require.Equal(t, "gid:0", regEntries.Entries[0].Selectors[0].Value)
	require.Equal(t, "uid:0", regEntries.Entries[0].Selectors[1].Value)
	require.Equal(t, "spiffe://shasta/test/uan1/workload2", regEntries.Entries[1].SpiffeId)
	require.Equal(t, "spiffe://shasta/uan/tenant1/uan1", regEntries.Entries[1].ParentId)
	require.Equal(t, "gid:0", regEntries.Entries[1].Selectors[0].Value)
	require.Equal(t, "uid:0", regEntries.Entries[1].Selectors[1].Value)
	require.Equal(t, int32(634000), regEntries.Entries[1].Ttl)
	require.Equal(t, "spiffe://shasta/test/uan2/workload1", regEntries.Entries[2].SpiffeId)
	require.Equal(t, "spiffe://shasta/uan/tenant1/uan2", regEntries.Entries[2].ParentId)
	require.Equal(t, "gid:0", regEntries.Entries[2].Selectors[0].Value)
	require.Equal(t, "uid:0", regEntries.Entries[2].Selectors[1].Value)
	require.Equal(t, "spiffe://shasta/test/uan2/workload2", regEntries.Entries[3].SpiffeId)
	require.Equal(t, "spiffe://shasta/uan/tenant1/uan2", regEntries.Entries[3].ParentId)
	require.Equal(t, "gid:0", regEntries.Entries[3].Selectors[0].Value)
	require.Equal(t, "uid:0", regEntries.Entries[3].Selectors[1].Value)
	require.Equal(t, int32(634000), regEntries.Entries[3].Ttl)
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

func TestCreateTPMWorkloadsXNAME(t *testing.T) {
	ds := fakedatastore.New(t)
	ctx := context.Background()

	os.Setenv("ENABLE_XNAME_WORKLOADS", "true")

	td, err := spiffeid.TrustDomainFromString("spiffe://shasta")
	if err != nil {
		t.Errorf("Failed to create trust domain: %v", err)
	}

	c := fakeentryclient.New(t, td, ds, nil)
	xnames := []string{"uan1", "uan2"}
	workloads := []tokens.Workload{
		{
			SpiffeID: "/test/XNAME/workload1",
			Selectors: []tokens.WorkloadSelector{
				{Type: "unix", Value: "uid:0"},
				{Type: "unix", Value: "gid:0"},
			},
		},
		{
			SpiffeID: "/test/XNAME/workload2",
			Selectors: []tokens.WorkloadSelector{
				{Type: "unix", Value: "uid:0"},
				{Type: "unix", Value: "gid:0"},
			},
			Ttl: 634000,
		},
	}

	for _, xname := range xnames {
		err := tokens.CreateTPMWorkloads(ctx, c, xname, workloads, "uan")
		if err != nil {
			t.Errorf("Failed to create registration record: %v", err)
		}
	}

	regEntries, err := ds.ListRegistrationEntries(ctx, &datastore.ListRegistrationEntriesRequest{})
	if err != nil {
		t.Errorf("Failed to request registration entries: %v", err)
	}

	require.Equal(t, "spiffe://shasta/test/uan1/workload1", regEntries.Entries[0].SpiffeId)
	require.Equal(t, "spiffe://shasta/uan/tenant1/uan1", regEntries.Entries[0].ParentId)
	require.Equal(t, "gid:0", regEntries.Entries[0].Selectors[0].Value)
	require.Equal(t, "uid:0", regEntries.Entries[0].Selectors[1].Value)
	require.Equal(t, "spiffe://shasta/test/uan1/workload2", regEntries.Entries[1].SpiffeId)
	require.Equal(t, "spiffe://shasta/uan/tenant1/uan1", regEntries.Entries[1].ParentId)
	require.Equal(t, "gid:0", regEntries.Entries[1].Selectors[0].Value)
	require.Equal(t, "uid:0", regEntries.Entries[1].Selectors[1].Value)
	require.Equal(t, int32(634000), regEntries.Entries[1].Ttl)
	require.Equal(t, "spiffe://shasta/test/uan2/workload1", regEntries.Entries[2].SpiffeId)
	require.Equal(t, "spiffe://shasta/uan/tenant1/uan2", regEntries.Entries[2].ParentId)
	require.Equal(t, "gid:0", regEntries.Entries[2].Selectors[0].Value)
	require.Equal(t, "uid:0", regEntries.Entries[2].Selectors[1].Value)
	require.Equal(t, "spiffe://shasta/test/uan2/workload2", regEntries.Entries[3].SpiffeId)
	require.Equal(t, "spiffe://shasta/uan/tenant1/uan2", regEntries.Entries[3].ParentId)
	require.Equal(t, "gid:0", regEntries.Entries[3].Selectors[0].Value)
	require.Equal(t, "uid:0", regEntries.Entries[3].Selectors[1].Value)
	require.Equal(t, int32(634000), regEntries.Entries[3].Ttl)
}

func TestCreateTPMWorkloads(t *testing.T) {
	ds := fakedatastore.New(t)
	ctx := context.Background()

	td, err := spiffeid.TrustDomainFromString("spiffe://shasta")
	if err != nil {
		t.Errorf("Failed to create trust domain: %v", err)
	}

	c := fakeentryclient.New(t, td, ds, nil)
	xnames := []string{"uan1", "uan2"}
	workloads := []tokens.Workload{
		{
			SpiffeID: "/test/workload1",
			Selectors: []tokens.WorkloadSelector{
				{Type: "unix", Value: "uid:0"},
				{Type: "unix", Value: "gid:0"},
			},
		},
		{
			SpiffeID: "/test/workload2",
			Selectors: []tokens.WorkloadSelector{
				{Type: "unix", Value: "uid:0"},
				{Type: "unix", Value: "gid:0"},
			},
			Ttl: 634000,
		},
	}

	for _, xname := range xnames {
		err := tokens.CreateTPMWorkloads(ctx, c, xname, workloads, "uan")
		if err != nil {
			t.Errorf("Failed to create registration record: %v", err)
		}
	}

	regEntries, err := ds.ListRegistrationEntries(ctx, &datastore.ListRegistrationEntriesRequest{})
	if err != nil {
		t.Errorf("Failed to request registration entries: %v", err)
	}

	require.Equal(t, "spiffe://shasta/test/workload1", regEntries.Entries[0].SpiffeId)
	require.Equal(t, "spiffe://shasta/uan/tenant1/uan1", regEntries.Entries[0].ParentId)
	require.Equal(t, "gid:0", regEntries.Entries[0].Selectors[0].Value)
	require.Equal(t, "uid:0", regEntries.Entries[0].Selectors[1].Value)

	require.Equal(t, "spiffe://shasta/test/workload1", regEntries.Entries[1].SpiffeId)
	require.Equal(t, "spiffe://shasta/uan/tenant1/uan2", regEntries.Entries[1].ParentId)
	require.Equal(t, "gid:0", regEntries.Entries[1].Selectors[0].Value)
	require.Equal(t, "uid:0", regEntries.Entries[1].Selectors[1].Value)

	require.Equal(t, "spiffe://shasta/test/workload2", regEntries.Entries[2].SpiffeId)
	require.Equal(t, "spiffe://shasta/uan/tenant1/uan1", regEntries.Entries[2].ParentId)
	require.Equal(t, "gid:0", regEntries.Entries[2].Selectors[0].Value)
	require.Equal(t, "uid:0", regEntries.Entries[2].Selectors[1].Value)
	require.Equal(t, int32(634000), regEntries.Entries[2].Ttl)

	require.Equal(t, "spiffe://shasta/test/workload2", regEntries.Entries[3].SpiffeId)
	require.Equal(t, "spiffe://shasta/uan/tenant1/uan2", regEntries.Entries[3].ParentId)
	require.Equal(t, "gid:0", regEntries.Entries[3].Selectors[0].Value)
	require.Equal(t, "uid:0", regEntries.Entries[3].Selectors[1].Value)
	require.Equal(t, int32(634000), regEntries.Entries[3].Ttl)
}

package tokens

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	entryv1 "github.com/spiffe/spire-api-sdk/proto/spire/api/server/entry/v1"
	"github.com/spiffe/spire-api-sdk/proto/spire/api/types"
)

func GenerateTPMWorkloads(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	ctx := context.Background()

	xname := r.FormValue("xname")

	serverType := r.FormValue("type")

	ec, err := NewEntryClient(socketPath, ctx)
	if err != nil {
		problem := ProblemDetails{
			Title:  "Error creating entry client",
			Status: http.StatusInternalServerError,
			Detail: fmt.Sprint(err.Error()),
		}
		log.Printf("Error: %s, Detail: %s", problem.Title, problem.Detail)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(problem)

		return
	}

	var workloadFile string

	switch serverType {
	case "compute":
		workloadFile = "/workloads/compute.yaml"
	case "ncn":
		workloadFile = "/workloads/ncn.yaml"
	case "storage":
		workloadFile = "/workloads/storage.yaml"
	case "uan":
		workloadFile = "/workloads/uan.yaml"
	default:
		problem := ProblemDetails{
			Title:  "invalid node type",
			Status: http.StatusInternalServerError,
			Detail: fmt.Sprintf("%s is not a valid node type", serverType),
		}
		log.Printf("Error: %s, Detail: %s", problem.Title, problem.Detail)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(problem)

		return
	}

	workloads, err := ParseWorkloads(workloadFile)
	if err != nil {
		problem := ProblemDetails{
			Title:  "Error Reading Workloads Configuration file",
			Status: http.StatusInternalServerError,
			Detail: fmt.Sprint(err.Error()),
		}
		log.Printf("Error: %s, Detail: %s", problem.Title, problem.Detail)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(problem)

		return
	}

	err = CreateTPMWorkloads(ctx, ec, xname, workloads, serverType)

	if err != nil {
		problem := ProblemDetails{
			Title:  "Error Reading Workloads Configuration file",
			Status: http.StatusInternalServerError,
			Detail: fmt.Sprint(err.Error()),
		}
		log.Printf("Error: %s, Detail: %s", problem.Title, problem.Detail)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(problem)

		return
	}
}

func CreateTPMWorkloads(ctx context.Context, c entryv1.EntryClient, xname string, workloads []Workload, serverType string) error {
	for _, workload := range workloads {
		var workloadID string
		m := regexp.MustCompile("^(.*)XNAME(.*)$")
		if strings.ToLower(os.Getenv("ENABLE_XNAME_WORKLOADS")) == "true" {
			workloadID = m.ReplaceAllString(workload.SpiffeID, "${1}"+xname+"${2}")
		} else {
			workloadID = workload.SpiffeID
		}

		selectors := []*types.Selector{}
		for _, selector := range workload.Selectors {
			selectors = append(selectors, &types.Selector{Type: selector.Type, Value: selector.Value})
		}
		selectors = append(selectors, &types.Selector{Type: "tpm_devid", Value: fmt.Sprintf("subject:cn:%s:%s", serverType, xname)})

		parentID := "/" + serverType + "/tenant1/" + xname

		var req *entryv1.BatchCreateEntryRequest
		if workload.Ttl != 0 {
			req = &entryv1.BatchCreateEntryRequest{
				Entries: []*types.Entry{
					{
						ParentId:  &types.SPIFFEID{TrustDomain: os.Getenv("SPIRE_DOMAIN"), Path: parentID},
						SpiffeId:  &types.SPIFFEID{TrustDomain: os.Getenv("SPIRE_DOMAIN"), Path: workloadID},
						Selectors: selectors,
						Ttl:       workload.Ttl,
					},
				},
			}
		} else {
			req = &entryv1.BatchCreateEntryRequest{
				Entries: []*types.Entry{
					{
						ParentId:  &types.SPIFFEID{TrustDomain: os.Getenv("SPIRE_DOMAIN"), Path: parentID},
						SpiffeId:  &types.SPIFFEID{TrustDomain: os.Getenv("SPIRE_DOMAIN"), Path: workloadID},
						Selectors: selectors,
					},
				},
			}
		}

		resp, err := c.BatchCreateEntry(ctx, req)

		// This needs to change if we expand to create more records at once.
		if resp.Results[0].Status.Message != "OK" {
			log.Printf("BatchCreateEntry Response: %v", resp)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

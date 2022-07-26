// Copyright 2021 Hewlett Packard Enterprise Development LP

package tokens

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	agentv1 "github.com/spiffe/spire-api-sdk/proto/spire/api/server/agent/v1"
	entryv1 "github.com/spiffe/spire-api-sdk/proto/spire/api/server/entry/v1"
	"github.com/spiffe/spire-api-sdk/proto/spire/api/types"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v2"
)

type WorkloadSelector struct {
	Type  string `yaml:"type"`
	Value string `yaml:"value"`
}
type Workload struct {
	SpiffeID  string             `yaml:"spiffeID"`
	Selectors []WorkloadSelector `yaml:"selectors"`
	Ttl       int32              `yaml:"ttl,omitempty"`
}

const (
	// Workload API (SPIRE default socket is assumed)
	socketPath = "/tmp/spire-server/private/api.sock"

	// optional timeout for the client context
	timeout = 5 * time.Second
)

// GenerateToken - generate new Spire Join token
func GenerateToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	ctx := context.Background()

	xname := r.FormValue("xname")
	serverType := r.FormValue("type")

	if serverType == "" {
		serverType = "compute"
	}

	log.Printf("Starting token generation with xname: %s", xname)

	ttl, err := strconv.Atoi(os.Getenv("TTL"))
	if err != nil {
		ttl = 60
	}

	if invalidID(xname) {
		problem := ProblemDetails{
			Title:  "Invalid or no xname provided",
			Status: http.StatusNotFound,
			Detail: fmt.Sprint("Xname must be alphanumeric"),
		}
		log.Printf("Error: %s, Detail: %s", problem.Title, problem.Detail)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(problem)
		return
	}

	ac, err := NewAgentClient(socketPath, ctx)
	if err != nil {
		problem := ProblemDetails{
			Title:  "Error creating agent client",
			Status: http.StatusInternalServerError,
			Detail: fmt.Sprint(err.Error()),
		}
		log.Printf("Error: %s, Detail: %s", problem.Title, problem.Detail)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(problem)

		return
	}

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

	token, err := CreateToken(ctx, ac, ttl)
	if err != nil {
		problem := ProblemDetails{
			Title:  "Error generating token",
			Status: http.StatusInternalServerError,
			Detail: fmt.Sprint(err.Error()),
		}
		log.Printf("Error: %s, Detail: %s", problem.Title, problem.Detail)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(problem)

		return
	}

	var spiffe_id string
	if serverType == "ncn" {
		spiffe_id = fmt.Sprintf("%s/%s", os.Getenv("NCN_ENTRY"), xname)
	} else if serverType == "storage" {
		spiffe_id = fmt.Sprintf("%s/%s", os.Getenv("STORAGE_ENTRY"), xname)
	} else if serverType == "uan" {
		spiffe_id = fmt.Sprintf("%s/%s", os.Getenv("UAN_ENTRY"), xname)
	} else {
		spiffe_id = fmt.Sprintf("%s/%s", os.Getenv("COMPUTE_ENTRY"), xname)
	}

	node_parent := fmt.Sprintf("/spire/agent/join_token/%s", token)
	err = CreateRegistrationRecord(ctx, ec, node_parent, spiffe_id)

	if err != nil {
		problem := ProblemDetails{
			Title:  "Error creating node registration record",
			Status: http.StatusInternalServerError,
			Detail: fmt.Sprint(err.Error()),
		}
		log.Printf("Error: %s, Detail: %s", problem.Title, problem.Detail)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(problem)

		return
	}

	log.Printf("Registration record generated with spiffeID: %s", spiffe_id)

	var cluster_entry string
	var workload_file string
	if serverType == "ncn" {
		cluster_entry = fmt.Sprintf("/%s", os.Getenv("NCN_CLUSTER_ENTRY"))
		workload_file = "/workloads/ncn.yaml"
	} else if serverType == "storage" {
		cluster_entry = fmt.Sprintf("/%s", os.Getenv("STORAGE_CLUSTER_ENTRY"))
		workload_file = "/workloads/storage.yaml"
	} else if serverType == "uan" {
		cluster_entry = fmt.Sprintf("/%s", os.Getenv("UAN_CLUSTER_ENTRY"))
		workload_file = "/workloads/uan.yaml"
	} else {
		cluster_entry = fmt.Sprintf("/%s", os.Getenv("COMPUTE_CLUSTER_ENTRY"))
		workload_file = "/workloads/compute.yaml"
	}

	err = CreateRegistrationRecord(ctx, ec, spiffe_id, cluster_entry) // cluster record

	if err != nil {
		problem := ProblemDetails{
			Title:  "Error creating cluster registration record",
			Status: http.StatusInternalServerError,
			Detail: fmt.Sprint(err.Error()),
		}
		log.Printf("Error: %s, Detail: %s", problem.Title, problem.Detail)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(problem)

		return
	}

	if strings.ToLower(os.Getenv("ENABLE_XNAME_WORKLOADS")) == "true" {
		log.Printf("Creating xname workload entries for %s", xname)

		workloads, err := ParseWorkloads(workload_file)
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

		err = CreateWorkloads(ctx, ec, xname, workloads, serverType)

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

	log.Printf("Cluster record generated with spiffeID: %s", cluster_entry)

	w.WriteHeader(http.StatusCreated)

	response := Token{
		JoinToken: token,
	}

	json.NewEncoder(w).Encode(response)
}

func invalidID(id string) bool {
	validID := regexp.MustCompile(`^[a-zA-Z0-9]*$`)

	if id == "" {
		return true
	}

	return !validID.MatchString(id)
}

func CreateToken(ctx context.Context, c agentv1.AgentClient, ttl int) (string, error) {
	req := &agentv1.CreateJoinTokenRequest{Ttl: int32(ttl)}
	resp, err := c.CreateJoinToken(ctx, req)
	if err != nil {
		return "", err
	}

	return resp.Value, nil
}

func CreateRegistrationRecord(ctx context.Context, c entryv1.EntryClient, parentID, spiffeID string) error {
	req := &entryv1.BatchCreateEntryRequest{
		Entries: []*types.Entry{
			{
				ParentId: &types.SPIFFEID{TrustDomain: os.Getenv("SPIRE_DOMAIN"), Path: parentID},
				SpiffeId: &types.SPIFFEID{TrustDomain: os.Getenv("SPIRE_DOMAIN"), Path: spiffeID},
				Selectors: []*types.Selector{
					{Type: "spiffe_id", Value: "spiffe://" + os.Getenv("SPIRE_DOMAIN") + parentID},
				},
			},
		},
	}
	resp, err := c.BatchCreateEntry(ctx, req)

	// This needs to change if we expand to create more records at once.
	if resp.Results[0].Status.Message != "OK" {
		log.Printf("BatchCreateEntry Response: %v", resp)
	}

	if err != nil {
		return err
	}

	return nil
}

func CreateWorkloads(ctx context.Context, c entryv1.EntryClient, xname string, workloads []Workload, serverType string) error {
	for _, workload := range workloads {
		m := regexp.MustCompile("^(.*)XNAME(.*)$")
		workloadID := m.ReplaceAllString(workload.SpiffeID, "${1}"+xname+"${2}")

		selectors := []*types.Selector{}
		for _, selector := range workload.Selectors {
			selectors = append(selectors, &types.Selector{Type: selector.Type, Value: selector.Value})
		}

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

func ParseWorkloads(file string) ([]Workload, error) {
	yamlFile, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	workloads := []Workload{}
	err = yaml.Unmarshal(yamlFile, &workloads)
	if err != nil {
		return nil, err
	}

	return workloads, nil
}

func NewAgentClient(socketPath string, ctx context.Context) (agentv1.AgentClient, error) {
	conn, err := grpc.Dial(socketPath, grpc.WithInsecure(), grpc.WithDialer(dialer)) //nolint: staticcheck
	if err != nil {
		return nil, err
	}
	return agentv1.NewAgentClient(conn), err
}

func NewEntryClient(socketPath string, ctx context.Context) (entryv1.EntryClient, error) {
	conn, err := grpc.Dial(socketPath, grpc.WithInsecure(), grpc.WithDialer(dialer)) //nolint: staticcheck
	if err != nil {
		return nil, err
	}
	return entryv1.NewEntryClient(conn), err
}

func dialer(addr string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("unix", addr, timeout)
}

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

	"github.com/spiffe/spire/pkg/common/idutil"
	"github.com/spiffe/spire/proto/spire/api/registration"
	"github.com/spiffe/spire/proto/spire/common"
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
	socketPath = "/tmp/spire-registration.sock"

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

	c, err := NewRegistrationClient(socketPath, ctx)
	if err != nil {
		problem := ProblemDetails{
			Title:  "Error creating registration client",
			Status: http.StatusInternalServerError,
			Detail: fmt.Sprint(err.Error()),
		}
		log.Printf("Error: %s, Detail: %s", problem.Title, problem.Detail)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(problem)

		return
	}

	token, err := CreateToken(ctx, c, ttl)
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
		spiffe_id = fmt.Sprintf("spiffe://%s%s/%s", os.Getenv("SPIRE_DOMAIN"), os.Getenv("NCN_ENTRY"), xname)
	} else if serverType == "storage" {
		spiffe_id = fmt.Sprintf("spiffe://%s%s/%s", os.Getenv("SPIRE_DOMAIN"), os.Getenv("STORAGE_ENTRY"), xname)
	} else {
		spiffe_id = fmt.Sprintf("spiffe://%s%s/%s", os.Getenv("SPIRE_DOMAIN"), os.Getenv("COMPUTE_ENTRY"), xname)
	}

	node_parent := fmt.Sprintf("spiffe://%s/spire/agent/join_token/%s", os.Getenv("SPIRE_DOMAIN"), token)
	err = CreateRegistrationRecord(ctx, c, node_parent, spiffe_id)

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
		cluster_entry = fmt.Sprintf("spiffe://%s%s", os.Getenv("SPIRE_DOMAIN"), os.Getenv("NCN_CLUSTER_ENTRY"))
		workload_file = "/workloads/ncn.yaml"
	} else if serverType == "storage" {
		cluster_entry = fmt.Sprintf("spiffe://%s%s", os.Getenv("SPIRE_DOMAIN"), os.Getenv("STORAGE_CLUSTER_ENTRY"))
		workload_file = "/workloads/storage.yaml"
	} else {
		cluster_entry = fmt.Sprintf("spiffe://%s%s", os.Getenv("SPIRE_DOMAIN"), os.Getenv("COMPUTE_CLUSTER_ENTRY"))
		workload_file = "/workloads/compute.yaml"
	}

	err = CreateRegistrationRecord(ctx, c, spiffe_id, cluster_entry) // cluster record

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

		err = CreateWorkloads(ctx, c, xname, workloads, serverType)

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

	return
}

func invalidID(id string) bool {

	var validID = regexp.MustCompile(`^[a-zA-Z0-9]*$`)

	if id == "" {
		return true
	}
	return !validID.MatchString(id)
}

func CreateToken(ctx context.Context, c registration.RegistrationClient, ttl int) (string, error) {
	req := &registration.JoinToken{Ttl: int32(ttl)}
	resp, err := c.CreateJoinToken(ctx, req)
	if err != nil {
		return "", err
	}

	return resp.Token, nil
}

func CreateRegistrationRecord(ctx context.Context, c registration.RegistrationClient, parentID, spiffeID string) error {
	id, err := idutil.ParseSpiffeID(spiffeID, idutil.AllowAnyTrustDomainWorkload())
	if err != nil {
		return err
	}

	req := &common.RegistrationEntry{
		ParentId: parentID,
		SpiffeId: id.String(),
		Selectors: []*common.Selector{
			{Type: "spiffe_id", Value: parentID},
		},
	}

	_, err = c.CreateEntryIfNotExists(ctx, req)
	if err != nil {
		return err
	}
	return nil
}

func CreateWorkloads(ctx context.Context, c registration.RegistrationClient, xname string, workloads []Workload, serverType string) error {

	for _, workload := range workloads {
		m := regexp.MustCompile("^(.*)XNAME(.*)$")
		workloadID := m.ReplaceAllString(workload.SpiffeID, "${1}"+xname+"${2}")

		selectors := []*common.Selector{}
		for _, selector := range workload.Selectors {
			selectors = append(selectors, &common.Selector{Type: selector.Type, Value: selector.Value})
		}

		var req *common.RegistrationEntry
		if workload.Ttl != 0 {
			req = &common.RegistrationEntry{
				ParentId:  "spiffe://shasta/" + serverType + "/tenant1/" + xname,
				SpiffeId:  workloadID,
				Selectors: selectors,
				Ttl:       workload.Ttl,
			}
		} else {
			req = &common.RegistrationEntry{
				ParentId:  "spiffe://shasta/" + serverType + "/tenant1/" + xname,
				SpiffeId:  workloadID,
				Selectors: selectors,
			}

		}

		_, err := c.CreateEntryIfNotExists(ctx, req)
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

func NewRegistrationClient(socketPath string, ctx context.Context) (registration.RegistrationClient, error) {
	conn, err := grpc.Dial(socketPath, grpc.WithInsecure(), grpc.WithDialer(dialer)) //nolint: staticcheck
	if err != nil {
		return nil, err
	}
	return registration.NewRegistrationClient(conn), err
}

func dialer(addr string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("unix", addr, timeout)
}

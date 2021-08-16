// Copyright 2021 Hewlett Packard Enterprise Development LP

package tokens

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

// RootGet - Returns service/api info
func RootGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	version, err := ioutil.ReadFile("/tokens-version")
	if err != nil {
		problem := ProblemDetails{
			Title:  "Error getting version",
			Status: http.StatusInternalServerError,
			Detail: fmt.Sprint(err),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(problem)
		return
	}
	info := Info{
		Version: strings.TrimSpace(string(version)),
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(info)
}

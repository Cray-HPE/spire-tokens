// Copyright 2020 Hewlett Packard Enterprise Development LP

package tokens

// ProblemDetails is a structure for returning an error via the api
type ProblemDetails struct {
	Title  string `json:"title,omitempty"`
	Status int32  `json:"status,omitempty"`
	Detail string `json:"detail,omitempty"`
}

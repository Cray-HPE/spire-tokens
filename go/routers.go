// Copyright 2020 Hewlett Packard Enterprise Development LP

package tokens

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// Route is an http route
type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

// Routes are a collection of http routes
type Routes []Route

// NewRouter creates a new router
func NewRouter() *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	for _, route := range routes {
		var handler http.Handler
		handler = route.HandlerFunc
		handler = Logger(handler, route.Name)

		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)
	}

	return router
}

var routes = Routes{
	{
		"RootGet",
		strings.ToUpper("Get"),
		"/api",
		RootGet,
	},
	{
		"ClientPost",
		strings.ToUpper("Post"),
		"/api/token",
		GenerateToken,
	},
}

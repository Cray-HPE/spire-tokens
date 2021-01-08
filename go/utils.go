// Copyright 2020 Hewlett Packard Enterprise Development LP

package tokens

// IntSliceContains will check for an integer value in a slice
func IntSliceContains(s []int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

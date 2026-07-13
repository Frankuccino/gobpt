package utils

import (
	"encoding/json"
	"fmt"
)

// This function is used to see the content of a struct, map, or slice to instantly take a look.
func PrintJSON(v interface{}) {
	json, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		fmt.Println("Error marshalling to JSON:", err)
	}
	fmt.Println("JSON", string(json))
}

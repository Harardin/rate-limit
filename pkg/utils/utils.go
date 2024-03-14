package utils

import (
	"math/rand"

	"github.com/goccy/go-json"
)

// Pointer return pointer for value
func Pointer[T any](v T) *T {
	return &v
}

func ExistInArray[T comparable](arr []T, value T) bool {
	for _, v := range arr {
		if v == value {
			return true
		}
	}
	return false
}

func GetRandomInt(min, max int) int {
	return rand.Intn(max-min) + min
}

// JsonToStruct return populates the fields of the dst struct from the fields
// of the src struct using json tags
func JsonToStruct(src interface{}, dst interface{}) error {
	result, err := json.Marshal(src)
	if err != nil {
		return err
	}

	return json.Unmarshal(result, dst)
}

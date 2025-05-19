package utils

import (
	"encoding/json"
	"reflect"
)

// Performs a deep copy of a struct or map
func DeepCopy(src interface{}) (interface{}, error) {
	if src == nil {
		return nil, nil
	}

	// For simple types, return as is
	v := reflect.ValueOf(src)
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Bool, reflect.String:
		return src, nil
	}

	// For complex types, use JSON marshaling
	// This is a compromise between efficiency and correctness
	data, err := json.Marshal(src)
	if err != nil {
		return nil, err
	}

	// Create a new instance of the same type as the source
	dst := reflect.New(reflect.TypeOf(src)).Interface()

	err = json.Unmarshal(data, dst)
	if err != nil {
		return nil, err
	}

	// If the source was a pointer, return the pointer
	// Otherwise, dereference the pointer
	if reflect.TypeOf(src).Kind() == reflect.Ptr {
		return dst, nil
	}
	return reflect.ValueOf(dst).Elem().Interface(), nil
}

// Performs a deep copy of an ansible.Task
func DeepCopyTask(src interface{}) (interface{}, error) {
	return DeepCopy(src)
}

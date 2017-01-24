// NOTICE: This is a derivative work of https://github.com/cloudfoundry/cli/blob/27a6d92bbcc298f73713983f0f798f3621ef8b1d/util/generic/merge_reduce.go
package manifest

import (
	"fmt"
	"reflect"
)

func DeepMerge(maps ...map[string]interface{}) map[string]interface{} {
	mergedmap := make(map[string]interface{})
	return Reduce(maps, mergedmap, mergeReducer)
}

func mergeReducer(key string, val interface{}, reduced map[string]interface{}) map[string]interface{} {

	switch {

	case containsKey(reduced, key) == false:
		reduced[key] = val
		return reduced

	case IsMappable(val):
		maps := []map[string]interface{}{Mappify(reduced[key].(interface{})), Mappify(val)}
		mergedmap := Reduce(maps, make(map[string]interface{}), mergeReducer)
		reduced[key] = mergedmap
		return reduced

	case IsSliceable(val):
		reduced[key] = append(reduced[key].([]interface{}), val.([]interface{})...)
		return reduced

	default:
		reduced[key] = val
		return reduced
	}
}

func containsKey(thing map[string]interface{}, key string) bool {
	if _, ok := thing[key]; ok {
		return true
	}
	return false
}

func IsSliceable(value interface{}) bool {
	if value == nil {
		return false
	}
	return reflect.TypeOf(value).Kind() == reflect.Slice
}

func IsMappable(value interface{}) bool {
	if value == nil {
		return false
	}
	switch value.(type) {
	case map[string]interface{}:
		return true
	default:
		return reflect.TypeOf(value).Kind() == reflect.Map
	}
}

func arrayMappify(data []interface{}) map[string]interface{} {
	if len(data) == 0 {
		return make(map[string]interface{})
	} else if len(data) > 1 {
		panic("Mappify called with more than one argument")
	}

	switch data := data[0].(type) {
	case nil:
		return make(map[string]interface{})
	case map[string]string:
		stringToInterfaceMap := make(map[string]interface{})

		for key, val := range data {
			stringToInterfaceMap[key] = val
		}
		return stringToInterfaceMap
	case map[string]interface{}:
		return data
	}

	panic("Mappify called with unexpected argument")
}

func Mappify(data interface{}) map[string]interface{} {

	switch data.(type) {
	case nil:
		return make(map[string]interface{})
		// TODO fix this commented code
	// case map[string]string:
	// 	stringToInterfaceMap := make(map[string]interface{})

	// 	for key, val := range data.(map[string]string) {
	// 		stringToInterfaceMap[key] = val
	// 	}
	// 	return stringToInterfaceMap
	// case map[string]interface{}:
	// 	return data.(map[string]interface{})
	case map[interface{}]interface{}:
		stringToInterfaceMap := make(map[string]interface{})

		for key, val := range data.(map[interface{}]interface{}) {
			stringedKey := fmt.Sprintf("%v", key)
			stringToInterfaceMap[stringedKey] = val
		}
		return stringToInterfaceMap

	}
	panic("Mappify called with unexpected argument")
}

type Reducer func(key string, val interface{}, reducedVal map[string]interface{}) map[string]interface{}

func Reduce(collections []map[string]interface{}, resultVal map[string]interface{}, cb Reducer) map[string]interface{} {
	for _, collection := range collections {
		for key := range collection {
			resultVal = cb(key, collection[key], resultVal)
		}
	}
	return resultVal
}

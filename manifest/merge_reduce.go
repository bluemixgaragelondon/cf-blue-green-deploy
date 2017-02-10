// NOTICE: This is a derivative work of https://github.com/cloudfoundry/cli/blob/27a6d92bbcc298f73713983f0f798f3621ef8b1d/util/generic/merge_reduce.go
package manifest

import (
	"fmt"
	"reflect"
)

func DeepMerge(maps ...map[string]interface{}) (map[string]interface{}, error) {
	mergedmap := make(map[string]interface{})
	return Reduce(maps, mergedmap, mergeReducer)
}

func mergeReducer(key string, val interface{}, reduced map[string]interface{}) (map[string]interface{}, error) {

	switch {

	case containsKey(reduced, key) == false:
		reduced[key] = val
		return reduced, nil

	case IsMappable(val):
		mVal, err := Mappify(val)
		if err != nil {
			return nil, err
		}
		mReduced, err := Mappify(reduced[key].(interface{}))
		if err != nil {
			return nil, err
		}

		maps := []map[string]interface{}{mReduced, mVal}
		mergedmap, err := Reduce(maps, make(map[string]interface{}), mergeReducer)
		if err != nil {
			return nil, err
		}
		reduced[key] = mergedmap
		return reduced, nil

	case IsSliceable(val):
		reduced[key] = append(reduced[key].([]interface{}), val.([]interface{})...)
		return reduced, nil

	default:
		reduced[key] = val
		return reduced, nil
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

func Mappify(data interface{}) (map[string]interface{}, error) {

	switch data.(type) {
	case nil:
		return make(map[string]interface{}), nil
	case map[string]string:
		stringToInterfaceMap := make(map[string]interface{})

		for key, val := range data.(map[string]string) {
			stringToInterfaceMap[key] = val
		}
		return stringToInterfaceMap, nil
	case map[string]interface{}:
		return data.(map[string]interface{}), nil
	case map[interface{}]interface{}:
		stringToInterfaceMap := make(map[string]interface{})

		for key, val := range data.(map[interface{}]interface{}) {
			stringedKey := fmt.Sprintf("%v", key)
			stringToInterfaceMap[stringedKey] = val
		}
		return stringToInterfaceMap, nil

	}
	return nil, fmt.Errorf("Mappify called with unexpected argument of type %T, expected map", data)
}

type Reducer func(key string, val interface{}, reducedVal map[string]interface{}) (map[string]interface{}, error)

func Reduce(collections []map[string]interface{}, resultVal map[string]interface{}, cb Reducer) (map[string]interface{}, error) {
	var err error
	for _, collection := range collections {
		for key := range collection {
			resultVal, err = cb(key, collection[key], resultVal)
			if err != nil {
				return nil, err
			}
		}
	}
	return resultVal, nil
}

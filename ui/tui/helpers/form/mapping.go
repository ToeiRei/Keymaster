// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package form

import (
	"errors"
	"reflect"
)

const mapTag string = "form"

func mapFromStruct(input any) (map[string]any, error) {
	v := reflect.Indirect(reflect.ValueOf(input))
	if v.Kind() != reflect.Struct {
		return nil, errors.New("mapFromStruct requires a struct or a pointer to a struct")
	}

	t := v.Type()
	result := make(map[string]any)

	for i := range t.NumField() {
		field := t.Field(i)

		if field.PkgPath != "" {
			continue
		}

		tagValue, ok := field.Tag.Lookup(mapTag)
		if !ok {
			continue
		}

		result[tagValue] = v.Field(i).Interface()
	}

	return result, nil
}

func mapToStruct[T any](input map[string]any, result *T) error {
	resultV := reflect.Indirect(reflect.ValueOf(result))
	if resultV.Kind() != reflect.Struct {
		return errors.New("mapToStruct requires a pointer to a struct")
	}

	resultT := resultV.Type()

	for i := range resultT.NumField() {
		resultF := resultT.Field(i)
		resultFV := resultV.Field(i)

		if !resultFV.CanSet() {
			continue
		}

		tagValue, ok := resultF.Tag.Lookup(mapTag)
		if !ok {
			continue
		}

		inputValue, ok := input[tagValue]
		if !ok {
			continue
		}

		inputValueV := reflect.ValueOf(inputValue)

		if inputValueV.Type().AssignableTo(resultFV.Type()) {
			resultFV.Set(inputValueV)
		} else if inputValueV.Type().ConvertibleTo(resultFV.Type()) {
			resultFV.Set(inputValueV.Convert(resultFV.Type()))
		}
	}

	return nil
}

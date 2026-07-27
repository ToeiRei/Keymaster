// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package form

import (
	"errors"
	"fmt"
	"reflect"
)

const mapTag string = "form"

// mapFrom flattens a form model into item values keyed by item id. A struct
// contributes its "form"-tagged fields, a string keyed map its entries.
func mapFrom(input any) (map[string]any, error) {
	v := reflect.Indirect(reflect.ValueOf(input))

	switch v.Kind() {
	case reflect.Struct:
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
	case reflect.Map:
		if v.Type().Key().Kind() != reflect.String {
			return nil, errors.New("mapFrom requires a map with string keys")
		}

		result := make(map[string]any, v.Len())
		for iter := v.MapRange(); iter.Next(); {
			result[iter.Key().String()] = iter.Value().Interface()
		}

		return result, nil
	default:
		return nil, errors.New("mapFrom requires a struct or a string keyed map, or a pointer to one")
	}
}

// mapTo is the inverse of [mapFrom]. Values that fit no field or key are
// dropped, as are valueless elements (buttons, labels, ...) reporting nil.
func mapTo[T any](input map[string]any, result *T) error {
	resultV := reflect.Indirect(reflect.ValueOf(result))

	switch resultV.Kind() {
	case reflect.Struct:
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

			if inputValueV, ok := convertTo(inputValue, resultFV.Type()); ok {
				resultFV.Set(inputValueV)
			}
		}

		return nil
	case reflect.Map:
		resultT := resultV.Type()
		if resultT.Key().Kind() != reflect.String {
			return errors.New("mapTo requires a map with string keys")
		}

		if resultV.IsNil() {
			if !resultV.CanSet() {
				return errors.New("mapTo cannot allocate the target map")
			}
			resultV.Set(reflect.MakeMapWithSize(resultT, len(input)))
		}

		for key, inputValue := range input {
			if inputValueV, ok := convertTo(inputValue, resultT.Elem()); ok {
				resultV.SetMapIndex(reflect.ValueOf(key).Convert(resultT.Key()), inputValueV)
			}
		}

		return nil
	default:
		return errors.New("mapTo requires a pointer to a struct or a string keyed map")
	}
}

// convertTo coerces an element value to target, reporting false when it does
// not fit. A nil value never fits, so valueless elements contribute nothing.
func convertTo(value any, target reflect.Type) (reflect.Value, bool) {
	v := reflect.ValueOf(value)

	switch {
	case !v.IsValid():
		return reflect.Value{}, false
	case v.Type().AssignableTo(target):
		return v, true
	case target.Kind() == reflect.String:
		// not Convert: that turns an integer into its rune, not its digits
		return reflect.ValueOf(fmt.Sprint(value)).Convert(target), true
	case v.Type().ConvertibleTo(target):
		return v.Convert(target), true
	}

	return reflect.Value{}, false
}

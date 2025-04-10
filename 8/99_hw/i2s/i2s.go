package main

import (
	"errors"
	"fmt"
	"reflect"
)

/*
func i2s(data interface{}, out interface{}) error {
	// todo
}
*/

func i2s(data interface{}, out interface{}) error {
	// Проверяем, что out — это указатель
	outVal := reflect.ValueOf(out)
	if outVal.Kind() != reflect.Ptr || outVal.IsNil() {
		return errors.New("output must be a non-nil pointer")
	}

	return fillValue(data, outVal.Elem())
}

func fillValue(data interface{}, val reflect.Value) error {
	// Проверка типа значения
	if !val.CanSet() {
		return errors.New("value is not settable")
	}

	switch val.Kind() {
	case reflect.Struct:
		// data должен быть map[string]interface{}
		mapData, ok := data.(map[string]interface{})
		if !ok {
			return fmt.Errorf("expected map for struct, got %T", data)
		}
		t := val.Type()
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			fieldVal := val.Field(i)
			if !fieldVal.CanSet() {
				continue
			}
			rawVal, ok := mapData[field.Name]
			if !ok {
				continue
			}
			err := fillValue(rawVal, fieldVal)
			if err != nil {
				return fmt.Errorf("field %s: %w", field.Name, err)
			}
		}
	case reflect.Slice:
		// data должен быть []interface{}
		sliceData, ok := data.([]interface{})
		if !ok {
			return fmt.Errorf("expected slice, got %T", data)
		}
		elemType := val.Type().Elem()
		sliceVal := reflect.MakeSlice(val.Type(), len(sliceData), len(sliceData))
		for i := 0; i < len(sliceData); i++ {
			elemPtr := reflect.New(elemType).Elem()
			err := fillValue(sliceData[i], elemPtr)
			if err != nil {
				return fmt.Errorf("slice index %d: %w", i, err)
			}
			sliceVal.Index(i).Set(elemPtr)
		}
		val.Set(sliceVal)
	case reflect.Int:
		floatVal, ok := data.(float64)
		if !ok {
			return fmt.Errorf("expected float64 for int, got %T", data)
		}
		val.SetInt(int64(floatVal))
	case reflect.String:
		strVal, ok := data.(string)
		if !ok {
			return fmt.Errorf("expected string, got %T", data)
		}
		val.SetString(strVal)
	case reflect.Bool:
		boolVal, ok := data.(bool)
		if !ok {
			return fmt.Errorf("expected bool, got %T", data)
		}
		val.SetBool(boolVal)
	default:
		return fmt.Errorf("unsupported kind: %s", val.Kind())
	}
	return nil
}

package querybuilder

import (
	"fmt"
	"reflect"
	"strings"
)

func InsertModel(table string, model any, suffix string) (string, []any, error) {
	cols, vals, err := columnsAndValuesFromModel(model)
	if err != nil {
		return "", nil, err
	}
	return InsertInto(table).
		Columns(cols...).
		Values(vals...).
		Suffix(suffix).
		ToSQL()
}

func columnsAndValuesFromModel(model any) ([]string, []any, error) {
	value := reflect.ValueOf(model)
	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil, nil, fmt.Errorf("model cannot be nil")
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return nil, nil, fmt.Errorf("model must be struct")
	}

	typ := value.Type()
	cols := make([]string, 0, typ.NumField())
	vals := make([]any, 0, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.PkgPath != "" {
			continue
		}
		tag := strings.TrimSpace(field.Tag.Get("db"))
		if tag == "" || tag == "-" {
			continue
		}
		col := strings.TrimSpace(strings.Split(tag, ",")[0])
		if col == "" || col == "-" {
			continue
		}
		cols = append(cols, col)
		vals = append(vals, value.Field(i).Interface())
	}

	if len(cols) == 0 {
		return nil, nil, fmt.Errorf("model has no db columns")
	}
	return cols, vals, nil
}

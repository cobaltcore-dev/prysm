// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0
package rgwadmin

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"
)

const queryAdminPath = "/admin"

// buildQueryPath constructs an API query path with the given parameters.
func buildQueryPath(endpoint, path, args string) string {
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	return fmt.Sprintf("%s%s%s%s%s", endpoint, queryAdminPath, path, separator, args)
}

// valueToURLParams encodes a struct into URL query parameters.
func valueToURLParams(i interface{}, acceptableFields []string) url.Values {
	values := url.Values{}
	values.Add("format", "json")

	// Convert acceptableFields to a map for fast lookups
	allowed := make(map[string]struct{}, len(acceptableFields))
	for _, field := range acceptableFields {
		if field != "" { // Ensure empty fields are ignored
			allowed[field] = struct{}{}
		}
	}

	populateURLParams(i, allowed, &values)
	return values
}

// addToURLParams appends struct values into an existing URL query map.
func addToURLParams(v *url.Values, i interface{}, acceptableFields []string) {
	allowed := make(map[string]struct{}, len(acceptableFields))
	for _, field := range acceptableFields {
		if field != "" { // Ensure empty fields are ignored
			allowed[field] = struct{}{}
		}
	}
	populateURLParams(i, allowed, v)
}

// populateURLParams extracts struct fields and adds them to URL parameters.
func populateURLParams(i interface{}, allowedFields map[string]struct{}, values *url.Values) {
	v := reflect.ValueOf(i)
	t := reflect.TypeOf(i)

	// Ensure we're working with a struct pointer
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return
		}
		v = v.Elem()
		t = t.Elem()
	}

	if v.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < v.NumField(); i++ {
		fieldValue := v.Field(i)
		fieldType := t.Field(i)
		tag := fieldType.Tag.Get("url")

		// Skip fields explicitly marked as "-" or if tag is empty
		if tag == "-" || tag == "" {
			continue
		}

		tagParts := strings.Split(tag, ",")
		name := tagParts[0]
		if name == "" {
			name = fieldType.Name
		}

		// Ensure the field is in the allowed list
		if _, ok := allowedFields[name]; !ok {
			continue
		}

		// Handle different data types and ensure no empty values are added
		switch fieldValue.Kind() {
		case reflect.String:
			if fieldValue.String() != "" {
				values.Add(name, fieldValue.String())
			}

		case reflect.Bool:
			values.Add(name, fmt.Sprint(fieldValue.Bool()))

		case reflect.Int, reflect.Int64, reflect.Int32:
			if fieldValue.Int() != 0 {
				values.Add(name, fmt.Sprint(fieldValue.Int()))
			}

		case reflect.Ptr:
			if fieldValue.IsValid() && !fieldValue.IsNil() {
				values.Add(name, fmt.Sprint(fieldValue.Elem()))
			}

		case reflect.Slice:
			for j := 0; j < fieldValue.Len(); j++ {
				item := fieldValue.Index(j)
				if item.IsValid() {
					values.Add(name, fmt.Sprint(item))
				}
			}

		case reflect.Struct:
			populateURLParams(fieldValue.Interface(), allowedFields, values)
		}
	}
}

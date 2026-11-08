package registry011

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

// The local preparation format has an exact schema. In particular, decoding
// through map[string]any would lose duplicate keys and can round uint64 values.
// Validate tokens against the destination types before the ordinary typed
// decode; do not normalize the policy's original canonical bytes.
func decodeHistoryPreparationInput(ctx context.Context, raw []byte) (historyPreparationInput, error) {
	var out historyPreparationInput
	if err := validateHistoryPreparationJSON(ctx, raw, reflect.TypeFor[historyPreparationInput]()); err != nil {
		return out, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&out); err != nil {
		return historyPreparationInput{}, ErrConfiguration
	}
	if err := ctx.Err(); err != nil {
		return historyPreparationInput{}, err
	}
	return out, nil
}

func validateHistoryPreparationJSON(ctx context.Context, raw []byte, schema reflect.Type) error {
	if ctx == nil || schema == nil || len(raw) == 0 || !utf8.Valid(raw) {
		return ErrConfiguration
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := historyPreparationJSONValue(ctx, decoder, schema, 0); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return ErrConfiguration
	}
	return ctx.Err()
}

func historyPreparationJSONValue(ctx context.Context, d *json.Decoder, schema reflect.Type, depth int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if depth > 64 {
		return ErrConfiguration
	}
	// The existing policy parser owns the policy schema and canonical form.
	// Still reject duplicate keys and excessive nesting in its raw subtree.
	if schema == reflect.TypeFor[json.RawMessage]() {
		schema = nil
	}
	token, err := d.Token()
	if err != nil {
		return ErrConfiguration
	}
	if opening, ok := token.(json.Delim); ok {
		switch opening {
		case '{':
			return historyPreparationJSONObject(ctx, d, schema, depth)
		case '[':
			return historyPreparationJSONArray(ctx, d, schema, depth)
		default:
			return ErrConfiguration
		}
	}
	if schema == nil {
		return nil // Raw policy values are checked again by its original parser.
	}
	switch schema.Kind() {
	case reflect.String:
		if _, ok := token.(string); ok {
			return nil
		}
	case reflect.Bool:
		if _, ok := token.(bool); ok {
			return nil
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if number, ok := token.(json.Number); ok {
			value, err := strconv.ParseInt(string(number), 10, schema.Bits())
			if err == nil && strconv.FormatInt(value, 10) == string(number) {
				return nil
			}
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if number, ok := token.(json.Number); ok {
			value, err := strconv.ParseUint(string(number), 10, schema.Bits())
			if err == nil && strconv.FormatUint(value, 10) == string(number) {
				return nil
			}
		}
	}
	return ErrConfiguration
}

func historyPreparationJSONObject(ctx context.Context, d *json.Decoder, schema reflect.Type, depth int) error {
	fields := make(map[string]reflect.Type)
	if schema != nil {
		if schema.Kind() != reflect.Struct {
			return ErrConfiguration
		}
		for i := range schema.NumField() {
			field := schema.Field(i)
			if field.PkgPath != "" {
				continue
			}
			name := strings.Split(field.Tag.Get("json"), ",")[0]
			if name == "-" {
				continue
			}
			if name == "" {
				name = field.Name
			}
			if _, duplicate := fields[name]; duplicate || field.Anonymous {
				return ErrConfiguration
			}
			fields[name] = field.Type
		}
	}
	seen := make(map[string]bool)
	for d.More() {
		if err := ctx.Err(); err != nil {
			return err
		}
		token, err := d.Token()
		key, ok := token.(string)
		if err != nil || !ok || seen[key] {
			return ErrConfiguration
		}
		seen[key] = true
		child, known := fields[key]
		if schema != nil && !known {
			return ErrConfiguration
		}
		if err := historyPreparationJSONValue(ctx, d, child, depth+1); err != nil {
			return err
		}
	}
	closing, err := d.Token()
	if err != nil || closing != json.Delim('}') || (schema != nil && len(seen) != len(fields)) {
		return ErrConfiguration
	}
	return nil
}

func historyPreparationJSONArray(ctx context.Context, d *json.Decoder, schema reflect.Type, depth int) error {
	var child reflect.Type
	if schema != nil {
		if schema.Kind() != reflect.Array {
			return ErrConfiguration
		}
		child = schema.Elem()
	}
	count := 0
	for d.More() {
		if schema != nil && count >= schema.Len() {
			return ErrConfiguration
		}
		if err := historyPreparationJSONValue(ctx, d, child, depth+1); err != nil {
			return err
		}
		count++
	}
	closing, err := d.Token()
	if err != nil || closing != json.Delim(']') || (schema != nil && count != schema.Len()) {
		return ErrConfiguration
	}
	return nil
}

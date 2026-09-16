package sessionv2

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
)

func NewID(prefix string) (string, error) {
	buf := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", err
	}
	prefix = strings.Trim(strings.TrimSpace(prefix), "-_")
	if prefix == "" {
		return hex.EncodeToString(buf), nil
	}
	return prefix + "_" + hex.EncodeToString(buf), nil
}

func CanonicalJSON(value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	if len(data) > MaxRecordBytes {
		return nil, fmt.Errorf("record exceeds %d-byte limit", MaxRecordBytes)
	}
	return data, nil
}

func DigestRecord(value any) (string, error) {
	copyValue := reflect.New(reflect.TypeOf(value))
	copyValue.Elem().Set(reflect.ValueOf(value))
	elem := copyValue.Elem()
	if elem.Kind() == reflect.Struct {
		field := elem.FieldByName("Digest")
		if field.IsValid() && field.CanSet() && field.Kind() == reflect.String {
			field.SetString("")
		}
	}
	data, err := CanonicalJSON(copyValue.Elem().Interface())
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// DecodeStrict rejects duplicate object keys, unknown fields, trailing values,
// oversized input, and non-integer JSON numbers before decoding into out.
func DecodeStrict(data []byte, out any) error {
	if len(data) > MaxRecordBytes {
		return fmt.Errorf("record exceeds %d-byte limit", MaxRecordBytes)
	}
	if err := rejectAmbiguousJSON(data); err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	dec.UseNumber()
	if err := dec.Decode(out); err != nil {
		return err
	}
	if dec.More() {
		return fmt.Errorf("trailing JSON value")
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("trailing JSON value")
		}
		return err
	}
	return nil
}

func rejectAmbiguousJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	return scanJSONValue(dec)
}

func scanJSONValue(dec *json.Decoder) error {
	token, err := dec.Token()
	if err != nil {
		return err
	}
	if number, ok := token.(json.Number); ok {
		if strings.ContainsAny(number.String(), ".eE") {
			return fmt.Errorf("non-integer numeric value %s is not allowed", number)
		}
		return nil
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for dec.More() {
			keyToken, err := dec.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("object key must be a string")
			}
			if seen[key] {
				return fmt.Errorf("duplicate JSON key %q", key)
			}
			seen[key] = true
			if err := scanJSONValue(dec); err != nil {
				return err
			}
		}
		_, err = dec.Token()
		return err
	case '[':
		for dec.More() {
			if err := scanJSONValue(dec); err != nil {
				return err
			}
		}
		_, err = dec.Token()
		return err
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delim)
	}
}

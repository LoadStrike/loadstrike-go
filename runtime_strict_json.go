package loadstrike

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

func decodeStrictJSONObject(
	reader io.Reader,
	maximumBytes int64,
	source string,
	allowed map[string]struct{},
) (map[string]json.RawMessage, error) {
	content, err := io.ReadAll(io.LimitReader(reader, maximumBytes+1))
	if err != nil {
		return nil, fmt.Errorf(
			"read %s: %s",
			source,
			sanitizeRuntimeDiagnostic(err.Error()),
		)
	}
	if int64(len(content)) > maximumBytes {
		return nil, fmt.Errorf("%s exceeds the %d byte safety limit", source, maximumBytes)
	}

	decoder := json.NewDecoder(bytes.NewReader(content))
	token, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", source, err)
	}
	delimiter, ok := token.(json.Delim)
	if !ok || delimiter != '{' {
		return nil, fmt.Errorf("decode %s: top-level JSON value must be an object", source)
	}

	fields := make(map[string]json.RawMessage, len(allowed))
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("decode %s: %w", source, err)
		}
		name, ok := token.(string)
		if !ok {
			return nil, fmt.Errorf("decode %s: object field name is invalid", source)
		}
		if _, ok := allowed[name]; !ok {
			return nil, fmt.Errorf("decode %s: unknown field %q", source, name)
		}
		if _, duplicate := fields[name]; duplicate {
			return nil, fmt.Errorf("decode %s: duplicate field %q", source, name)
		}

		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, fmt.Errorf("decode %s field %q: %w", source, name, err)
		}
		fields[name] = append(json.RawMessage(nil), value...)
	}
	if _, err := decoder.Token(); err != nil {
		return nil, fmt.Errorf("decode %s: %w", source, err)
	}

	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("decode %s: trailing JSON content is not allowed", source)
		}
		return nil, fmt.Errorf("decode %s: invalid trailing content: %w", source, err)
	}
	return fields, nil
}

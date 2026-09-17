// Package schemas builds the JSON Schema files in this folder into the
// compiled program, so a service can check incoming messages against them
// without needing a copy of the .json files at runtime. The .json files
// remain the only definition of what a valid message is.
package schemas

import (
	"bytes"
	"embed"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed *.json
var files embed.FS

// Load compiles one schema file by name, e.g. "co2_reading.schema.json".
func Load(name string) (*jsonschema.Schema, error) {
	data, err := files.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("read schema %s: %w", name, err)
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("parse schema %s: %w", name, err)
	}
	c := jsonschema.NewCompiler()
	// "format" (e.g. date-time) is only a label by default; this makes the
	// compiler reject values that don't match it.
	c.AssertFormat()
	// Registered under the same URL as the file's own $id; a bare file name
	// would be resolved against the working directory and show up in error
	// messages as a misleading local path.
	url := "https://smart-ventilation.local/schemas/" + name
	if err := c.AddResource(url, doc); err != nil {
		return nil, fmt.Errorf("add schema %s: %w", name, err)
	}
	return c.Compile(url)
}

// Validate checks a raw JSON message against a schema.
func Validate(sch *jsonschema.Schema, payload []byte) error {
	msg, err := jsonschema.UnmarshalJSON(bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("not valid JSON: %w", err)
	}
	return sch.Validate(msg)
}

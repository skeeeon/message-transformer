package transformer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"text/template"
	"time"

	"go.uber.org/zap"
)

// Transformer handles message transformations
type Transformer struct {
	logger        *zap.Logger
	templateCache map[string]*template.Template
}

type TransformError struct {
	Message string
	Err     error
}

func (e *TransformError) Error() string {
	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

// New creates a new transformer
func New(logger *zap.Logger) *Transformer {
	return &Transformer{
		logger:        logger,
		templateCache: make(map[string]*template.Template),
	}
}

// Transform applies a template transformation to the input data
func (t *Transformer) Transform(templateStr string, inputData []byte) ([]byte, error) {
	// Parse input data into a map for template access
	var data map[string]interface{}
	decoder := json.NewDecoder(bytes.NewReader(inputData))
	decoder.UseNumber() // Use json.Number for numbers to preserve precision
	if err := decoder.Decode(&data); err != nil {
		return nil, &TransformError{
			Message: "failed to parse input data",
			Err:     err,
		}
	}

	// Get or create template
	tmpl, err := t.getTemplate(templateStr)
	if err != nil {
		return nil, &TransformError{
			Message: "failed to parse template",
			Err:     err,
		}
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, &TransformError{
			Message: "failed to execute template",
			Err:     err,
		}
	}

	// Validate output is valid JSON
	var output interface{}
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		return nil, &TransformError{
			Message: "template output is not valid JSON",
			Err:     err,
		}
	}

	t.logger.Debug("Message transformed successfully",
		zap.Any("input", data),
		zap.Any("output", output))

	return buf.Bytes(), nil
}

// getTemplate retrieves a cached template or creates a new one
func (t *Transformer) getTemplate(templateStr string) (*template.Template, error) {
	if tmpl, ok := t.templateCache[templateStr]; ok {
		return tmpl, nil
	}

	// Create new template with custom functions
	tmpl, err := template.New("transform").Funcs(template.FuncMap{
		"toJSON": func(v interface{}) string {
			b, err := json.Marshal(v)
			if err != nil {
				return "null"
			}
			return string(b)
		},
		"fromJSON": func(s string) interface{} {
			var v interface{}
			if err := json.Unmarshal([]byte(s), &v); err != nil {
				return nil
			}
			return v
		},
		"now": func() string {
			return time.Now().UTC().Format(time.RFC3339)
		},
		"num": func(v interface{}) string {
			switch n := v.(type) {
			case json.Number:
				return string(n)
			case float64:
				return strconv.FormatFloat(n, 'f', -1, 64)
			case float32:
				return strconv.FormatFloat(float64(n), 'f', -1, 32)
			case int:
				return strconv.Itoa(n)
			case int64:
				return strconv.FormatInt(n, 10)
			case int32:
				return strconv.FormatInt(int64(n), 10)
			case string:
				// Try to parse as number
				if _, err := strconv.ParseFloat(n, 64); err == nil {
					return n
				}
				return "0" // Default to 0 if not a valid number
			default:
				return "0" // Default to 0 for non-numeric types
			}
		},
		"bool": func(v interface{}) string {
			switch b := v.(type) {
			case bool:
				return strconv.FormatBool(b)
			case string:
				if b == "true" || b == "false" {
					return b
				}
				return "false"
			case int, int64, float64:
				return "true" // Non-zero numbers are true
			case nil:
				return "false"
			default:
				return "false"
			}
		},
	}).Parse(templateStr)

	if err != nil {
		return nil, err
	}

	t.templateCache[templateStr] = tmpl
	return tmpl, nil
}

// ClearCache clears the template cache
func (t *Transformer) ClearCache() {
	t.templateCache = make(map[string]*template.Template)
}

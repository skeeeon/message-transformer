//file: internal/transformer/transformer.go

package transformer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"text/template"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"message-transformer/internal/config"
	"message-transformer/internal/metrics"
)

// Transformer handles message transformations with pre-compiled templates
type Transformer struct {
	logger    *zap.Logger
	metrics   metrics.Recorder
	templates sync.Map // thread-safe map for template access
}

// CompiledTemplate wraps a pre-compiled template with metadata
type CompiledTemplate struct {
	Template *template.Template
	ID       string
}

// TransformError wraps transformation errors with context
type TransformError struct {
	Message string
	Err     error
}

func (e *TransformError) Error() string {
	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

// templateFuncs returns the common template functions
func templateFuncs() template.FuncMap {
	return template.FuncMap{
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
		"uuid7": func() string {
			id, err := uuid.NewV7()
			if err != nil {
				// Fallback to V4 if V7 generation fails
				id = uuid.New()
			}
			return id.String()
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
				if _, err := strconv.ParseFloat(n, 64); err == nil {
					return n
				}
				return "0"
			default:
				return "0"
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
				return "true"
			case nil:
				return "false"
			default:
				return "false"
			}
		},
	}
}

// New creates a new transformer with pre-compiled templates
func New(logger *zap.Logger, rules []config.Rule, metricsRecorder metrics.Recorder) (*Transformer, error) {
	if metricsRecorder == nil {
		metricsRecorder = metrics.NewNoOpRecorder()
	}

	t := &Transformer{
		logger:  logger,
		metrics: metricsRecorder,
	}

	// Pre-compile all templates at startup
	for _, rule := range rules {
		if err := t.compileTemplate(rule.ID, rule.Transform.Template); err != nil {
			return nil, fmt.Errorf("failed to compile template for rule %s: %w", rule.ID, err)
		}
	}

	// Record the number of active rules
	t.metrics.SetActiveRules(len(rules))

	return t, nil
}

// compileTemplate compiles a template and stores it in the sync.Map
func (t *Transformer) compileTemplate(id, templateStr string) error {
	tmpl, err := template.New(id).
		Funcs(templateFuncs()).
		Parse(templateStr)
	if err != nil {
		t.metrics.IncTemplateErrors(id)
		return err
	}

	t.templates.Store(id, &CompiledTemplate{
		Template: tmpl,
		ID:       id,
	})
	return nil
}

// Transform applies a pre-compiled template transformation to the input data
func (t *Transformer) Transform(ruleID string, inputData []byte) ([]byte, error) {
	start := time.Now()
	t.metrics.ObserveTransformInputSize(ruleID, int64(len(inputData)))

	// Get pre-compiled template
	tmplValue, exists := t.templates.Load(ruleID)
	if !exists {
		t.metrics.IncTransformErrors(ruleID)
		return nil, &TransformError{
			Message: "template not found",
			Err:     fmt.Errorf("no template for rule %s", ruleID),
		}
	}
	compiledTmpl := tmplValue.(*CompiledTemplate)

	// Parse input data using a decoder for precise number handling
	var data map[string]interface{}
	decoder := json.NewDecoder(bytes.NewReader(inputData))
	decoder.UseNumber()
	if err := decoder.Decode(&data); err != nil {
		t.metrics.IncTransformErrors(ruleID)
		return nil, &TransformError{
			Message: "failed to parse input data",
			Err:     err,
		}
	}

	// Execute template with buffer pool for efficiency
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)

	if err := compiledTmpl.Template.Execute(buf, data); err != nil {
		t.metrics.IncTemplateErrors(ruleID)
		return nil, &TransformError{
			Message: "failed to execute template",
			Err:     err,
		}
	}

	// Validate output is valid JSON
	output := buf.Bytes()
	if !json.Valid(output) {
		t.metrics.IncTransformErrors(ruleID)
		return nil, &TransformError{
			Message: "template output is not valid JSON",
			Err:     fmt.Errorf("invalid JSON output for rule %s", ruleID),
		}
	}

	// Create a copy of the output since we're returning the buffer to the pool
	result := make([]byte, len(output))
	copy(result, output)

	// Record metrics
	duration := time.Since(start).Seconds()
	t.metrics.ObserveTransformDuration(ruleID, duration)
	t.metrics.ObserveTransformOutputSize(ruleID, int64(len(result)))

	t.logger.Debug("Message transformed successfully",
		zap.String("rule_id", ruleID),
		zap.Int("input_size", len(inputData)),
		zap.Int("output_size", len(result)),
		zap.Float64("duration_ms", duration*1000))

	return result, nil
}

// Buffer pool for template execution
var bufPool = sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

// AddTemplate adds a new template at runtime (useful for dynamic rule updates)
func (t *Transformer) AddTemplate(id, templateStr string) error {
	err := t.compileTemplate(id, templateStr)
	if err != nil {
		t.metrics.IncTemplateErrors(id)
		return err
	}

	// Update active rules count
	count := 0
	t.templates.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	t.metrics.SetActiveRules(count)

	return nil
}

// RemoveTemplate removes a template (useful for rule cleanup)
func (t *Transformer) RemoveTemplate(id string) {
	t.templates.Delete(id)

	// Update active rules count
	count := 0
	t.templates.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	t.metrics.SetActiveRules(count)
}

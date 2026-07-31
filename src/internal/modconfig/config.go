// Package modconfig parses and validates a mod's config.yaml: a flat map
// of scalar key/value settings (string/number/boolean), deliberately
// without metadata (label/description/enum/secret/required) — the author
// writes this by hand, the client infers the widget from the YAML value's
// own type. See docs/manifest-schema.md.
package modconfig

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Config is a mod's config.yaml, decoded.
type Config map[string]any

// Parse decodes raw YAML into a flat Config, rejecting anything that isn't a
// top-level mapping of plain scalars — no nested maps/sequences. An absent
// or empty file has no bearing here: the caller decides whether config.yaml
// existing at all means "this mod has config", not this function.
func Parse(raw []byte) (Config, error) {
	var m map[string]any
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("invalid yaml: %w", err)
	}

	cfg := make(Config, len(m))
	for k, v := range m {
		switch v.(type) {
		case string, int, float64, bool, nil:
			cfg[k] = v
		default:
			return nil, fmt.Errorf("config.yaml key %q: value must be a plain string/number/boolean (no nested maps or lists)", k)
		}
	}
	return cfg, nil
}

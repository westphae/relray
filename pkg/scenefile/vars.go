package scenefile

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// VarRange represents a named variable sweeping linearly from Start to End.
type VarRange struct {
	Name       string
	Start, End float64
}

// varPattern matches $name or $name:default in YAML text.
// Group 1: variable name, Group 2 (optional): default value.
var varPattern = regexp.MustCompile(`\$([a-zA-Z_]\w*)(?::([+-]?(?:\d+\.?\d*|\.\d+)(?:[eE][+-]?\d+)?))?`)

// SubstituteVars replaces $name and $name:default tokens in raw YAML text.
// Provided vars override defaults. If a variable has no value and no default,
// it is left as-is (which will cause a YAML parse error).
func SubstituteVars(text []byte, vars map[string]float64) []byte {
	return varPattern.ReplaceAllFunc(text, func(match []byte) []byte {
		groups := varPattern.FindSubmatch(match)
		name := string(groups[1])

		if val, ok := vars[name]; ok {
			return []byte(strconv.FormatFloat(val, 'g', 10, 64))
		}
		if len(groups[2]) > 0 {
			// Use the default value as-is (it's already a valid number string)
			return groups[2]
		}
		// No value provided and no default — leave unchanged for clear error
		return match
	})
}

// ParseRange parses a "name:start:end" string into a VarRange.
func ParseRange(s string) (VarRange, error) {
	parts := strings.SplitN(s, ":", 3)
	if len(parts) != 3 {
		return VarRange{}, fmt.Errorf("invalid range %q: expected name:start:end", s)
	}
	start, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return VarRange{}, fmt.Errorf("invalid range %q: bad start value: %w", s, err)
	}
	end, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return VarRange{}, fmt.Errorf("invalid range %q: bad end value: %w", s, err)
	}
	return VarRange{Name: parts[0], Start: start, End: end}, nil
}

// ParseVar parses a "name=value" string into a variable name and value.
func ParseVar(s string) (string, float64, error) {
	parts := strings.SplitN(s, "=", 2)
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid var %q: expected name=value", s)
	}
	val, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return "", 0, fmt.Errorf("invalid var %q: bad value: %w", s, err)
	}
	return parts[0], val, nil
}

// InterpolateVars computes variable values at parameter t ∈ [0, 1].
func InterpolateVars(ranges []VarRange, t float64) map[string]float64 {
	vars := make(map[string]float64, len(ranges))
	for _, r := range ranges {
		vars[r.Name] = r.Start + t*(r.End-r.Start)
	}
	return vars
}

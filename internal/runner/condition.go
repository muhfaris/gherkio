package runner

import (
	"fmt"
	"strconv"
	"strings"
)

// EvaluateCondition parses and evaluates the given conditional expression against the variables map.
func EvaluateCondition(cond string, vars map[string]interface{}) (bool, error) {
	cond = strings.TrimSpace(cond)
	if cond == "" {
		return true, nil
	}

	// Check if it's negation shorthand e.g. "!$var"
	negate := false
	if strings.HasPrefix(cond, "!") {
		negate = true
		cond = strings.TrimSpace(cond[1:])
	}

	// Check for comparison operators in priority order
	var op string
	var leftStr, rightStr string
	operators := []string{"!=", "==", ">=", "<=", ">", "<"}
	for _, o := range operators {
		if idx := strings.Index(cond, o); idx >= 0 {
			op = o
			leftStr = cond[:idx]
			rightStr = cond[idx+len(o):]
			break
		}
	}

	if op != "" {
		if negate {
			return false, fmt.Errorf("cannot use negation '!' with comparison operator in condition %q", cond)
		}
		left, err := resolveOperand(leftStr, vars)
		if err != nil {
			return false, err
		}
		right, err := resolveOperand(rightStr, vars)
		if err != nil {
			return false, err
		}
		return compareValues(op, left, right)
	}

	// Shorthand truthiness check e.g. "$var" or "!$var"
	val, err := resolveOperand(cond, vars)
	if err != nil {
		return false, err
	}

	truthy := isTruthy(val)
	if negate {
		return !truthy, nil
	}
	return truthy, nil
}

// resolveOperand parses an operand from a condition string, resolving variables or generator calls if needed.
func resolveOperand(expr string, vars map[string]interface{}) (interface{}, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, nil
	}

	// Check if it's a variable reference e.g., "$var", "${var}" or "$accounts.eka.username"
	if strings.HasPrefix(expr, "$") {
		varName := expr[1:]
		if strings.HasPrefix(varName, "{") && strings.HasSuffix(varName, "}") {
			varName = varName[1 : len(varName)-1]
		}
		
		// If variable exists (supports dotted paths, array-index bracket notation)
		if val, ok := resolveNestedVar(varName, vars); ok {
			return val, nil
		}

		// Check if it's a generator call like $randomInt or ${randomInt(1,10)}
		// We can interpolate it to get the value.
		interpolated, err := interpolateString(expr, vars)
		if err == nil && interpolated != expr {
			return parseLiteral(interpolated), nil
		}
		return nil, nil // Treat undefined variables as nil
	}

	return parseLiteral(expr), nil
}

// parseLiteral converts a raw string operand into a typed value (string, float64, bool, nil).
func parseLiteral(s string) interface{} {
	// Strip double or single quotes if present
	if (strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"")) ||
		(strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'")) {
		return s[1 : len(s)-1]
	}

	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}
	if s == "null" || s == "nil" {
		return nil
	}

	// Try to parse as float64
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}

	return s
}

// compareValues compares two resolved operand values using the given operator.
func compareValues(op string, left, right interface{}) (bool, error) {
	if left == nil || right == nil {
		switch op {
		case "==":
			return left == right, nil
		case "!=":
			return left != right, nil
		default:
			return false, fmt.Errorf("cannot use operator %q with nil value", op)
		}
	}

	// If both can be numbers, compare numerically
	lf, lOk := toFloat64(left)
	rf, rOk := toFloat64(right)
	if lOk && rOk {
		switch op {
		case "==":
			return lf == rf, nil
		case "!=":
			return lf != rf, nil
		case ">":
			return lf > rf, nil
		case ">=":
			return lf >= rf, nil
		case "<":
			return lf < rf, nil
		case "<=":
			return lf <= rf, nil
		default:
			return false, fmt.Errorf("unknown operator %q for numeric comparison", op)
		}
	}

	// Otherwise, compare as strings or direct equality
	switch op {
	case "==":
		return fmt.Sprintf("%v", left) == fmt.Sprintf("%v", right), nil
	case "!=":
		return fmt.Sprintf("%v", left) != fmt.Sprintf("%v", right), nil
	case ">", ">=", "<", "<=":
		return false, fmt.Errorf("cannot compare non-numeric values %v and %v using operator %q", left, right, op)
	default:
		return false, fmt.Errorf("unknown operator %q", op)
	}
}

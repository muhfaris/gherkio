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
	return evaluateBooleanExpression(cond, vars)
}

func evaluateBooleanExpression(expr string, vars map[string]interface{}) (bool, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return false, fmt.Errorf("empty condition expression")
	}

	stripped, err := stripOuterParentheses(expr)
	if err != nil {
		return false, err
	}
	if stripped != expr {
		return evaluateBooleanExpression(stripped, vars)
	}

	parts, err := splitBooleanExpression(expr, "||")
	if err != nil {
		return false, err
	}
	if len(parts) > 1 {
		for _, part := range parts {
			matched, evalErr := evaluateBooleanExpression(part, vars)
			if evalErr != nil {
				return false, evalErr
			}
			if matched {
				return true, nil
			}
		}
		return false, nil
	}

	parts, err = splitBooleanExpression(expr, "&&")
	if err != nil {
		return false, err
	}
	if len(parts) > 1 {
		for _, part := range parts {
			matched, evalErr := evaluateBooleanExpression(part, vars)
			if evalErr != nil {
				return false, evalErr
			}
			if !matched {
				return false, nil
			}
		}
		return true, nil
	}

	if strings.HasPrefix(expr, "!") && !strings.HasPrefix(expr, "!=") {
		remainder := strings.TrimSpace(expr[1:])
		// Preserve the existing rule for ambiguous unparenthesized negated
		// comparisons. Use !($value == expected) to make intent explicit.
		if !strings.HasPrefix(remainder, "(") && containsComparison(remainder) {
			return false, fmt.Errorf("cannot use negation '!' with comparison operator in condition %q", remainder)
		}
		matched, evalErr := evaluateBooleanExpression(remainder, vars)
		return !matched, evalErr
	}

	return evaluateAtomicCondition(expr, vars)
}

func evaluateAtomicCondition(cond string, vars map[string]interface{}) (bool, error) {

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

	return isTruthy(val), nil
}

func containsComparison(expr string) bool {
	for _, operator := range []string{"!=", "==", ">=", "<=", ">", "<"} {
		if strings.Contains(expr, operator) {
			return true
		}
	}
	return false
}

// splitBooleanExpression splits only at top-level operators, ignoring quoted
// strings and parenthesized subexpressions.
func splitBooleanExpression(expr, operator string) ([]string, error) {
	var parts []string
	start, depth := 0, 0
	var quote rune
	runes := []rune(expr)
	for i := 0; i < len(runes); i++ {
		current := runes[i]
		if quote != 0 {
			if current == quote && (i == 0 || runes[i-1] != '\\') {
				quote = 0
			}
			continue
		}
		if current == '\'' || current == '"' {
			quote = current
			continue
		}
		switch current {
		case '(':
			depth++
		case ')':
			depth--
			if depth < 0 {
				return nil, fmt.Errorf("unmatched ')' in condition %q", expr)
			}
		}
		if depth == 0 && i+1 < len(runes) && string(runes[i:i+2]) == operator {
			part := strings.TrimSpace(string(runes[start:i]))
			if part == "" {
				return nil, fmt.Errorf("missing operand around %s in condition %q", operator, expr)
			}
			parts = append(parts, part)
			start = i + 2
			i++
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote in condition %q", expr)
	}
	if depth != 0 {
		return nil, fmt.Errorf("unmatched '(' in condition %q", expr)
	}
	if len(parts) == 0 {
		return []string{expr}, nil
	}
	last := strings.TrimSpace(string(runes[start:]))
	if last == "" {
		return nil, fmt.Errorf("missing operand around %s in condition %q", operator, expr)
	}
	return append(parts, last), nil
}

func stripOuterParentheses(expr string) (string, error) {
	if !strings.HasPrefix(expr, "(") {
		return expr, nil
	}
	depth := 0
	var quote rune
	runes := []rune(expr)
	for i, current := range runes {
		if quote != 0 {
			if current == quote && (i == 0 || runes[i-1] != '\\') {
				quote = 0
			}
			continue
		}
		if current == '\'' || current == '"' {
			quote = current
			continue
		}
		if current == '(' {
			depth++
		}
		if current == ')' {
			depth--
			if depth < 0 {
				return "", fmt.Errorf("unmatched ')' in condition %q", expr)
			}
			if depth == 0 && i != len(runes)-1 {
				return expr, nil
			}
		}
	}
	if quote != 0 {
		return "", fmt.Errorf("unterminated quote in condition %q", expr)
	}
	if depth != 0 {
		return "", fmt.Errorf("unmatched '(' in condition %q", expr)
	}
	return strings.TrimSpace(string(runes[1 : len(runes)-1])), nil
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

	// Also resolve bare dotted paths (e.g. "item.price") when they exist in vars.
	// This lets $if conditions reference scoped aliases without a $ prefix.
	if val, ok := resolveNestedVar(expr, vars); ok {
		return val, nil
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

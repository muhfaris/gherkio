package runner

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// StepLocation defines the line boundaries of a step within a scenario file.
type StepLocation struct {
	Index     int    // 0-indexed step index within the section
	Section   string // "setup", "steps", "teardown"
	StartLine int    // 1-indexed start line (inclusive)
	EndLine   int    // 1-indexed end line (inclusive)
}

var (
	sectionRegex   = regexp.MustCompile(`^\s*(setup|steps|teardown)\s*:`)
	stepStartRegex = regexp.MustCompile(`^\s*-\s`)
)

// LocateStep finds the step containing a specific line number within a Gherkio test file.
func LocateStep(filepath string, lineNum int) (*StepLocation, error) {
	steps, err := ScanSteps(filepath)
	if err != nil {
		return nil, err
	}

	for _, loc := range steps {
		if lineNum >= loc.StartLine && lineNum <= loc.EndLine {
			return &loc, nil
		}
	}

	return nil, fmt.Errorf("no step found containing line %d in %s", lineNum, filepath)
}

// ScanSteps parses a Gherkio test file line-by-line to detect step boundaries.
func ScanSteps(filepath string) ([]StepLocation, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var steps []StepLocation
	var currentSection string
	var activeStep *StepLocation
	var sectionStepCount map[string]int = make(map[string]int)
	var sectionIndent map[string]string = make(map[string]string)

	for idx, line := range lines {
		lineNum := idx + 1
		trimmed := strings.TrimSpace(line)

		// Check for section headers
		if matches := sectionRegex.FindStringSubmatch(line); len(matches) > 1 {
			// Close previous active step if it exists
			if activeStep != nil {
				activeStep.EndLine = lineNum - 1
				steps = append(steps, *activeStep)
				activeStep = nil
			}
			currentSection = matches[1]
			continue
		}

		// Check if we are inside a valid section and find a step start
		if currentSection != "" {
			if stepStartRegex.MatchString(line) {
				indent, ok := sectionIndent[currentSection]
				currentIndent := getLeadingWhitespace(line)
				if !ok {
					sectionIndent[currentSection] = currentIndent
					indent = currentIndent
					ok = true
				}

				if currentIndent == indent {
					// Close previous active step if it exists
					if activeStep != nil {
						activeStep.EndLine = lineNum - 1
						steps = append(steps, *activeStep)
					}

					// Start a new step
					idx := sectionStepCount[currentSection]
					sectionStepCount[currentSection]++

					activeStep = &StepLocation{
						Index:     idx,
						Section:   currentSection,
						StartLine: lineNum,
						EndLine:   lineNum, // Default to same line, will grow
					}
					continue
				}
			}

			if activeStep != nil {
				// Expand end line of active step to include non-empty lines, comments, etc.
				// (Empty lines at the end will be pruned or left to the step)
				if trimmed != "" {
					activeStep.EndLine = lineNum
				}
			}
		}
	}

	// Close the last active step
	if activeStep != nil {
		activeStep.EndLine = len(lines)
		steps = append(steps, *activeStep)
	}

	return steps, nil
}

func getLeadingWhitespace(line string) string {
	var ws strings.Builder
	for _, ch := range line {
		if ch == ' ' || ch == '\t' {
			ws.WriteRune(ch)
		} else {
			break
		}
	}
	return ws.String()
}

package llm

import (
	"regexp"
	"strings"
)

// ExtractCode removes markdown code blocks (e.g., ```python ... ```) and returns the raw code.
func ExtractCode(input string) string {
	// Regex to find content inside ```python ... ``` or ``` ... ```
	re := regexp.MustCompile("(?s)```(?:python)?\n?(.*?)\n?```")
	matches := re.FindStringSubmatch(input)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	// If no markdown blocks, return the trimmed input
	return strings.TrimSpace(input)
}

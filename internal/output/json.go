package output

import (
	"encoding/json"
	"fmt"
	"os"
)

// WriteJSON marshals v to stdout with trailing newline.
func WriteJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}

// WriteError writes a message to stderr.
func WriteError(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

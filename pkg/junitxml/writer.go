package junitxml

import (
	"encoding/xml"
	"fmt"
	"io"
)

func (testsuites JUnitTestSuites) Write(w io.Writer) error {
	if _, err := io.WriteString(w, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"); err != nil {
		return fmt.Errorf("failed to write XML header: %w", err)
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "\t")
	if err := enc.Encode(testsuites); err != nil {
		return fmt.Errorf("failed to encode XML document: %w", err)
	}
	if _, err := io.WriteString(w, "\n"); err != nil {
		return fmt.Errorf("failed to write XML footer: %w", err)
	}
	return nil
}

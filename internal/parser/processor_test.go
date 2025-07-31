package parser

import (
	"os"
	"strings"
	"testing"
)

func TestProcessFile(t *testing.T) {
	input := "msgid=1234:\n{}"
	p := NewProcessor()
	err := p.ProcessFile(strings.NewReader(input), 0, false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestProcessFile_zip(t *testing.T) {
	filename := "../../response-files/response-attachment-zip-password.log"
	file, err := os.Open(filename)
	if err != nil {
		t.Errorf("Cannot open file: %v", err)
	}
	defer file.Close()
	p := NewProcessor()
	err = p.ProcessFile(file, 0, false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

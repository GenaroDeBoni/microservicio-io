package service

import (
	"errors"
	"testing"

	"microservicio-io/internal/apperr"
)

type mockExtractor struct {
	text string
	err  error
}

func (m *mockExtractor) Extract(data []byte) (string, error) { return m.text, m.err }

func TestExtract_Success(t *testing.T) {
	svc := NewService(&mockExtractor{text: "hola mundo"}, 15<<20)
	got, err := svc.Extract([]byte("%PDF-1.4..."), "doc.pdf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hola mundo" {
		t.Errorf("Extract() = %q, want %q", got, "hola mundo")
	}
}

func TestExtract_EmptyInput(t *testing.T) {
	svc := NewService(&mockExtractor{}, 15<<20)
	_, err := svc.Extract(nil, "doc.pdf")
	if err == nil {
		t.Fatal("expected error for empty input, got nil")
	}
	if got := apperr.CodeOf(err); got != apperr.CodeInvalidFile {
		t.Errorf("code = %q, want %q", got, apperr.CodeInvalidFile)
	}
}

func TestExtract_TooLarge(t *testing.T) {
	svc := NewService(&mockExtractor{}, 5)
	big := make([]byte, 6)
	_, err := svc.Extract(big, "doc.pdf")
	if err == nil {
		t.Fatal("expected too-large error, got nil")
	}
	if got := apperr.CodeOf(err); got != apperr.CodeTooLarge {
		t.Errorf("code = %q, want %q", got, apperr.CodeTooLarge)
	}
}

func TestExtract_ExtractorError_Malformed(t *testing.T) {
	svc := NewService(&mockExtractor{err: errors.New("bad pdf")}, 15<<20)
	_, err := svc.Extract([]byte("%PDF-1.4..."), "doc.pdf")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := apperr.CodeOf(err); got != apperr.CodeMalformedPDF {
		t.Errorf("code = %q, want %q", got, apperr.CodeMalformedPDF)
	}
}

func TestExtract_NotAPDF(t *testing.T) {
	svc := NewService(&mockExtractor{}, 15<<20)
	_, err := svc.Extract([]byte("hello world"), "doc.pdf")
	if err == nil {
		t.Fatal("expected invalid-file error, got nil")
	}
	if got := apperr.CodeOf(err); got != apperr.CodeInvalidFile {
		t.Errorf("code = %q, want %q", got, apperr.CodeInvalidFile)
	}
}

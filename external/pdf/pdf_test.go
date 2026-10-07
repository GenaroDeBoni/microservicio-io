package pdf

import (
	"strings"
	"testing"
)

func TestIsPDF(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{"valid magic", []byte("%PDF-1.4 content"), true},
		{"empty", []byte{}, false},
		{"nil", nil, false},
		{"not pdf text", []byte("hello world"), false},
		{"truncated magic", []byte("%PD"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPDF(tt.data); got != tt.want {
				t.Errorf("IsPDF(%q) = %v, want %v", tt.data, got, tt.want)
			}
		})
	}
}

func TestExtract_NotPDF(t *testing.T) {
	_, err := Extract([]byte("not a pdf"))
	if err == nil {
		t.Fatal("expected error for non-PDF input")
	}
	if !strings.Contains(err.Error(), "PDF") {
		t.Errorf("error should mention PDF, got %q", err.Error())
	}
}

func TestExtract_ValidMinimalPDF(t *testing.T) {
	pdfBytes := minimalPDF("Hello World")
	text, err := Extract(pdfBytes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(text, "Hello World") {
		t.Errorf("extracted text = %q, want to contain %q", text, "Hello World")
	}
}

// minimalPDF genera un PDF mínimo con un solo texto en la página.
func minimalPDF(text string) []byte {
	content := "BT /F1 12 Tf 100 700 Td (" + text + ") Tj ET"
	var sb strings.Builder
	sb.WriteString("%PDF-1.4\n")
	offsets := make([]int, 0)
	add := func(s string) {
		offsets = append(offsets, sb.Len())
		sb.WriteString(s)
	}
	add("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")
	add("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")
	add("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>\nendobj\n")
	stream := "4 0 obj\n<< /Length " + itoa(len(content)) + " >>\nstream\n" + content + "\nendstream\nendobj\n"
	add(stream)
	add("5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n")

	xrefOffset := sb.Len()
	sb.WriteString("xref\n0 6\n0000000000 65535 f \n")
	for _, off := range offsets {
		sb.WriteString(pad10(off) + " 00000 n \n")
	}
	sb.WriteString("trailer\n<< /Size 6 /Root 1 0 R >>\nstartxref\n" + itoa(xrefOffset) + "\n%%EOF")
	return []byte(sb.String())
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func pad10(n int) string {
	s := itoa(n)
	for len(s) < 10 {
		s = "0" + s
	}
	return s
}

// Package pdf es el adaptador de infraestructura que extrae texto de PDFs
// usando github.com/ledongthuc/pdf.
package pdf

import (
	"bytes"
	"io"

	"github.com/ledongthuc/pdf"
)

// IsPDF valida que los bytes comienzan con los magic bytes de PDF (%PDF).
func IsPDF(data []byte) bool {
	return len(data) >= 5 && bytes.HasPrefix(data, []byte("%PDF"))
}

// Extract extrae el texto plano de los bytes de un PDF.
// Implementa service.ExtractorPort.
func Extract(data []byte) (string, error) {
	if !IsPDF(data) {
		return "", errNotPDF
	}

	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}

	plain, err := reader.GetPlainText()
	if err != nil {
		return "", err
	}

	text, err := io.ReadAll(plain)
	if err != nil {
		return "", err
	}
	return string(text), nil
}

type notPDFError struct{}

func (notPDFError) Error() string { return "los bytes no corresponden a un PDF válido" }

var errNotPDF = notPDFError{}

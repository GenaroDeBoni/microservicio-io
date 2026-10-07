// Package service implementa la lógica de negocio de extracción de texto PDF.
// Depende de ExtractorPort (puerto) para no acoplarse a una librería concreta (DIP).
package service

import (
	"bytes"

	"microservicio-io/internal/apperr"
)

const pdfMagic = "%PDF"

// ExtractorPort es el puerto que implementa el adaptador de extracción (DIP/SOLID).
type ExtractorPort interface {
	Extract(data []byte) (string, error)
}

// Service orquesta la validación y la extracción de texto.
type Service struct {
	extractor ExtractorPort
	maxSize   int64
}

// NewService crea un Service con el extractor y el límite de tamaño inyectados.
func NewService(extractor ExtractorPort, maxSize int64) *Service {
	return &Service{extractor: extractor, maxSize: maxSize}
}

// Extract valida la entrada y delega la extracción al adaptador.
func (s *Service) Extract(data []byte, filename string) (string, error) {
	if len(data) == 0 {
		return "", apperr.InvalidFile("No se recibieron bytes para procesar.")
	}
	if int64(len(data)) > s.maxSize {
		return "", apperr.TooLarge(s.maxSize / (1 << 20))
	}
	if !bytes.HasPrefix(data, []byte(pdfMagic)) {
		return "", apperr.InvalidFile("El archivo no es un PDF válido.")
	}

	text, err := s.extractor.Extract(data)
	if err != nil {
		return "", apperr.MalformedPDF("No se pudo extraer el texto: " + err.Error())
	}
	return text, nil
}

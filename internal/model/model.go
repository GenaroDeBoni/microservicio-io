// Package model contiene los DTO de entrada y salida del microservicio.
package model

// ExtractResponse es la respuesta exitosa de POST /api/v1/extract.
// Los campos coinciden con el contrato esperado por RemotePdfProcessor.
type ExtractResponse struct {
	Filename  string `json:"filename"`
	Extension string `json:"extension"`
	MimeType  string `json:"mime_type"`
	Text      string `json:"text"`
}

// ProblemDetail es el cuerpo de error según RFC 9457.
type ProblemDetail struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail"`
	Instance string `json:"instance"`
}

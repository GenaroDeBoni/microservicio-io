// Package handler define la capa de presentación HTTP del microservicio.
package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"microservicio-io/external/pdf"
	"microservicio-io/internal/apperr"
	"microservicio-io/internal/model"
)

const contentTypeProblem = "application/problem+json"

// Extractor es la dependencia del handler (puerto hacia la capa de servicio).
// El handler no conoce *service.Service: sólo esta interfaz (DIP).
type Extractor interface {
	Extract(data []byte, filename string) (string, error)
}

// Handler agrupa los handlers HTTP con sus dependencias inyectadas.
type Handler struct {
	svc Extractor
}

// NewHandler crea un Handler con la dependencia inyectada.
func NewHandler(svc Extractor) *Handler {
	return &Handler{svc: svc}
}

// Health responde GET /health con 200 {"status":"ok"} (liveness/readiness).
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Extract responde POST /api/v1/extract: recibe un PDF por multipart/form-data
// en el campo "file", extrae su texto y devuelve metadata + texto.
// Errores en formato RFC 9457 application/problem+json.
func (h *Handler) Extract(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		h.writeProblem(c, apperr.InvalidFile("El campo 'file' es requerido."))
		return
	}

	f, err := file.Open()
	if err != nil {
		h.writeProblem(c, apperr.Server("No se pudo leer el archivo subido."))
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		h.writeProblem(c, apperr.Server("No se pudo leer el archivo subido."))
		return
	}

	if !pdf.IsPDF(data) {
		h.writeProblem(c, apperr.InvalidFile("El archivo no es un PDF válido."))
		return
	}

	text, err := h.svc.Extract(data, file.Filename)
	if err != nil {
		h.writeProblem(c, err)
		return
	}

	c.JSON(http.StatusOK, model.ExtractResponse{
		Filename:  file.Filename,
		Extension: filepath.Ext(file.Filename),
		MimeType:  "application/pdf",
		Text:      text,
	})
}

// writeProblem serializa un error como RFC 9457 con el Content-Type correcto.
func (h *Handler) writeProblem(c *gin.Context, err error) {
	pd := apperr.ToProblemDetail(err, c.Request.URL.Path)
	body, _ := json.Marshal(pd)
	c.Data(pd.Status, contentTypeProblem, body)
}

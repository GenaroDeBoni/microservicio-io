// microservicio-io: microservicio de extracción de texto PDF.
//
// Recibe un PDF por multipart/form-data en POST /api/v1/extract y devuelve el
// texto extraído con metadata, según el contrato de RemotePdfProcessor del
// monolito. Errores en formato RFC 9457 application/problem+json.
package main

import (
	"log"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"

	"microservicio-io/external/pdf"
	"microservicio-io/internal/handler"
	"microservicio-io/internal/service"
)

const (
	defaultPort     = "8080"
	defaultMaxMB    = 15
	envPort         = "PORT"
	envMaxSizeMB    = "PDF_MAX_SIZE_MB"
)

// extractorFunc adapta una función al puerto ExtractorPort (DIP).
type extractorFunc func([]byte) (string, error)

func (f extractorFunc) Extract(data []byte) (string, error) { return f(data) }

func main() {
	port := getenv(envPort, defaultPort)
	maxSizeMB := getenvInt(envMaxSizeMB, defaultMaxMB)

	svc := service.NewService(extractorFunc(pdf.Extract), int64(maxSizeMB)<<20)
	h := handler.NewHandler(svc)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/health", h.Health)
	r.POST("/api/v1/extract", h.Extract)

	addr := ":" + port
	log.Printf("microservicio-io escuchando en %s (max PDF: %d MB)", addr, maxSizeMB)
	if err := r.Run(addr); err != nil {
		log.Fatalf("no se pudo iniciar el servidor: %v", err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
		log.Printf("valor inválido para %s=%q, usando default %d", key, v, fallback)
	}
	return fallback
}

package main

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"microservicio-io/external/pdf"
	"microservicio-io/internal/handler"
	"microservicio-io/internal/service"
)

// newTestServer arma el stack completo: pdf adapter → service → handler → gin.
func newTestServer(maxSizeMB int64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	svc := service.NewService(adapterFunc(pdf.Extract), maxSizeMB<<20)
	h := handler.NewHandler(svc)
	r := gin.New()
	r.GET("/health", h.Health)
	r.POST("/api/v1/extract", h.Extract)
	return r
}

type adapterFunc func([]byte) (string, error)

func (f adapterFunc) Extract(data []byte) (string, error) { return f(data) }

func TestIntegration_Health(t *testing.T) {
	r := newTestServer(15)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestIntegration_ExtractValidPDF(t *testing.T) {
	r := newTestServer(15)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", "doc.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(minimalPDF("Hola Mundo")); err != nil {
		t.Fatal(err)
	}
	mw.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/extract", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Filename  string `json:"filename"`
		Extension string `json:"extension"`
		MimeType  string `json:"mime_type"`
		Text      string `json:"text"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if resp.Filename != "doc.pdf" || resp.Extension != ".pdf" || resp.MimeType != "application/pdf" {
		t.Errorf("metadata incorrecta: %+v", resp)
	}
	if !strings.Contains(resp.Text, "Hola Mundo") {
		t.Errorf("text = %q, want to contain %q", resp.Text, "Hola Mundo")
	}
}

func TestIntegration_ExtractNonPDF_400ProblemDetail(t *testing.T) {
	r := newTestServer(15)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "doc.txt")
	fw.Write([]byte("no soy pdf"))
	mw.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/extract", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", ct)
	}
	var pd map[string]any
	json.Unmarshal(w.Body.Bytes(), &pd)
	if pd["type"] != "/invalid-file" {
		t.Errorf("type = %v, want /invalid-file", pd["type"])
	}
}

func TestIntegration_TooLarge_413(t *testing.T) {
	r := newTestServer(1) // límite de 1 MB

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "big.pdf")
	big := append([]byte("%PDF-1.4"), make([]byte, 2<<20)...)
	fw.Write(big)
	mw.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/extract", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	r.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413; body=%s", w.Code, w.Body.String())
	}
	var pd map[string]any
	json.Unmarshal(w.Body.Bytes(), &pd)
	if pd["type"] != "/too-large" {
		t.Errorf("type = %v, want /too-large", pd["type"])
	}
}

// minimalPDF genera un PDF mínimo con un texto en la primera página.
func minimalPDF(text string) []byte {
	content := "BT /F1 12 Tf 100 700 Td (" + text + ") Tj ET"
	var sb strings.Builder
	sb.WriteString("%PDF-1.4\n")
	var offsets []int
	add := func(s string) {
		offsets = append(offsets, sb.Len())
		sb.WriteString(s)
	}
	add("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")
	add("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")
	add("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>\nendobj\n")
	add("4 0 obj\n<< /Length " + itoa(len(content)) + " >>\nstream\n" + content + "\nendstream\nendobj\n")
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

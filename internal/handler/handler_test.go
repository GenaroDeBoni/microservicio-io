package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"microservicio-io/internal/apperr"
)

// stubExtractor implementa Extractor para tests (doble determinista).
type stubExtractor struct {
	text string
	err  error
}

func (s *stubExtractor) Extract(data []byte, filename string) (string, error) {
	return s.text, s.err
}

// newRouter monta las rutas con un extractor inyectado.
func newRouter(svc Extractor) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(svc)
	r.POST("/api/v1/extract", h.Extract)
	r.GET("/health", h.Health)
	return r
}

// multipartBody construye un body multipart con el campo dado.
func multipartBody(t *testing.T, fieldName, filename string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile(fieldName, filename)
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := fw.Write(content); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	return &buf, w.FormDataContentType()
}

func TestHealth(t *testing.T) {
	r := newRouter(&stubExtractor{})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status field = %v, want ok", body["status"])
	}
}

func TestExtract_Success(t *testing.T) {
	r := newRouter(&stubExtractor{text: "texto extraido"})
	body, ctype := multipartBody(t, "file", "doc.pdf", []byte("%PDF-1.4 test"))
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/extract", body)
	req.Header.Set("Content-Type", ctype)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if resp["text"] != "texto extraido" {
		t.Errorf("text = %v, want %q", resp["text"], "texto extraido")
	}
	if resp["filename"] != "doc.pdf" {
		t.Errorf("filename = %v, want doc.pdf", resp["filename"])
	}
	if resp["mime_type"] != "application/pdf" {
		t.Errorf("mime_type = %v, want application/pdf", resp["mime_type"])
	}
	if resp["extension"] != ".pdf" {
		t.Errorf("extension = %v, want .pdf", resp["extension"])
	}
}

func TestExtract_InvalidFile_NotPDF(t *testing.T) {
	r := newRouter(&stubExtractor{})
	body, ctype := multipartBody(t, "file", "doc.txt", []byte("hello"))
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/extract", body)
	req.Header.Set("Content-Type", ctype)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
	assertProblemDetail(t, w, 400, "/invalid-file")
}

func TestExtract_MissingFileField(t *testing.T) {
	r := newRouter(&stubExtractor{})
	body, ctype := multipartBody(t, "other", "doc.pdf", []byte("%PDF"))
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/extract", body)
	req.Header.Set("Content-Type", ctype)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
	assertProblemDetail(t, w, 400, "/invalid-file")
}

func TestExtract_MalformedPDF_422(t *testing.T) {
	r := newRouter(&stubExtractor{err: apperr.MalformedPDF("corrupto")})
	body, ctype := multipartBody(t, "file", "doc.pdf", []byte("%PDF-1.4..."))
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/extract", body)
	req.Header.Set("Content-Type", ctype)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422; body=%s", w.Code, w.Body.String())
	}
	assertProblemDetail(t, w, 422, "/malformed-pdf")
}

func TestExtract_TooLarge_413(t *testing.T) {
	r := newRouter(&stubExtractor{err: apperr.TooLarge(15)})
	body, ctype := multipartBody(t, "file", "big.pdf", []byte("%PDF-1.4 big"))
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/extract", body)
	req.Header.Set("Content-Type", ctype)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413; body=%s", w.Code, w.Body.String())
	}
	assertProblemDetail(t, w, 413, "/too-large")
}

func TestExtract_Timeout_504(t *testing.T) {
	r := newRouter(&stubExtractor{err: apperr.Timeout()})
	body, ctype := multipartBody(t, "file", "slow.pdf", []byte("%PDF-1.4 slow"))
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/extract", body)
	req.Header.Set("Content-Type", ctype)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusGatewayTimeout {
		t.Fatalf("status = %d, want 504; body=%s", w.Code, w.Body.String())
	}
	assertProblemDetail(t, w, 504, "/timeout")
}

func TestExtract_ServerError_500(t *testing.T) {
	r := newRouter(&stubExtractor{err: errors.New("boom")})
	body, ctype := multipartBody(t, "file", "doc.pdf", []byte("%PDF-1.4"))
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/extract", body)
	req.Header.Set("Content-Type", ctype)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", w.Code, w.Body.String())
	}
	assertProblemDetail(t, w, 500, "/server-error")
}

// assertProblemDetail verifica que la respuesta sea RFC 9457.
func assertProblemDetail(t *testing.T, w *httptest.ResponseRecorder, wantStatus int, wantType string) {
	t.Helper()
	if ct := w.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", ct)
	}
	var pd map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &pd); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if pd["type"] != wantType {
		t.Errorf("type = %v, want %q", pd["type"], wantType)
	}
	if pd["status"] != float64(wantStatus) {
		t.Errorf("status = %v, want %d", pd["status"], wantStatus)
	}
	for _, field := range []string{"type", "title", "status", "detail", "instance"} {
		if _, ok := pd[field]; !ok {
			t.Errorf("missing RFC 9457 field %q", field)
		}
	}
}

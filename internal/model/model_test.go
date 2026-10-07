package model

import (
	"encoding/json"
	"testing"
)

func TestExtractResponse_JSON(t *testing.T) {
	resp := ExtractResponse{
		Filename:  "doc.pdf",
		Extension: ".pdf",
		MimeType:  "application/pdf",
		Text:      "hola",
	}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	want := map[string]string{
		"filename":  "doc.pdf",
		"extension": ".pdf",
		"mime_type": "application/pdf",
		"text":      "hola",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("field %q = %v, want %q", k, got[k], v)
		}
	}
}

func TestProblemDetail_JSON(t *testing.T) {
	pd := ProblemDetail{
		Type:     "/invalid-file",
		Title:    "invalid-file",
		Status:   400,
		Detail:   "detalle",
		Instance: "/api/v1/extract",
	}
	b, err := json.Marshal(pd)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, f := range []string{"type", "title", "status", "detail", "instance"} {
		if _, ok := got[f]; !ok {
			t.Errorf("missing field %q in JSON", f)
		}
	}
}

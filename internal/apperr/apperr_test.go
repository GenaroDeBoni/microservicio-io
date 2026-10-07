package apperr

import "testing"

func TestAppError_Error(t *testing.T) {
	err := InvalidFile("el archivo no es PDF")
	if err.Error() != "el archivo no es PDF" {
		t.Errorf("Error() = %q, want %q", err.Error(), "el archivo no es PDF")
	}
}

func TestStatusFor(t *testing.T) {
	tests := []struct {
		code Code
		want int
	}{
		{CodeInvalidFile, 400},
		{CodeMalformedPDF, 422},
		{CodeTooLarge, 413},
		{CodeTimeout, 504},
		{CodeServer, 500},
	}
	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			if got := statusFor(tt.code); got != tt.want {
				t.Errorf("statusFor(%q) = %d, want %d", tt.code, got, tt.want)
			}
		})
	}
}

func TestToProblemDetail(t *testing.T) {
	tests := []struct {
		name     string
		err      *AppError
		instance string
		wantType string
		wantSt   int
	}{
		{"invalid-file", InvalidFile("no es pdf"), "/api/v1/extract", "/invalid-file", 400},
		{"malformed-pdf", MalformedPDF("corrupto"), "/api/v1/extract", "/malformed-pdf", 422},
		{"too-large", TooLarge(15), "/api/v1/extract", "/too-large", 413},
		{"timeout", Timeout(), "/api/v1/extract", "/timeout", 504},
		{"server-error", Server("boom"), "/api/v1/extract", "/server-error", 500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pd := ToProblemDetail(tt.err, tt.instance)
			if pd.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", pd.Type, tt.wantType)
			}
			if pd.Status != tt.wantSt {
				t.Errorf("Status = %d, want %d", pd.Status, tt.wantSt)
			}
			if pd.Title != string(tt.err.Code) {
				t.Errorf("Title = %q, want %q", pd.Title, tt.err.Code)
			}
			if pd.Detail != tt.err.Detail {
				t.Errorf("Detail = %q, want %q", pd.Detail, tt.err.Detail)
			}
			if pd.Instance != tt.instance {
				t.Errorf("Instance = %q, want %q", pd.Instance, tt.instance)
			}
		})
	}
}

func TestToProblemDetail_NonAppError(t *testing.T) {
	pd := ToProblemDetail(errGeneric{"x"}, "/health")
	if pd.Status != 500 || pd.Type != "/server-error" {
		t.Errorf("generic error should map to 500 /server-error, got %d %q", pd.Status, pd.Type)
	}
}

type errGeneric struct{ msg string }

func (e errGeneric) Error() string { return e.msg }

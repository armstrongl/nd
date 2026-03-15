package output

// JSONResponse is the standard envelope for all --json output.
type JSONResponse struct {
	Status string      `json:"status"`           // "ok", "error", "partial"
	Data   interface{} `json:"data,omitempty"`
	Errors []JSONError `json:"errors,omitempty"`
}

// JSONError represents a single error in JSON output.
type JSONError struct {
	Code    string `json:"code"`            // Machine-readable error code
	Message string `json:"message"`         // Human-readable message
	Field   string `json:"field,omitempty"` // Relevant field or path
}

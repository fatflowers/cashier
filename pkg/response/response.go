package response

// New generic response spec
type APIResponseCode int

const (
	APIResponseCodeOK         APIResponseCode = 0
	APIResponseCodeBadRequest APIResponseCode = 40000
	APIResponseCodeError      APIResponseCode = 50000
)

var codeToMsg = map[APIResponseCode]string{
	APIResponseCodeOK:         "ok",
	APIResponseCodeBadRequest: "unexpected error",
}

// APIResponse is the generic response envelope used by HTTP APIs.
// Use OKT / ErrorT helpers to construct instances.
type APIResponse[T any] struct {
	Code    APIResponseCode `json:"code"`
	Message string          `json:"message"`
	Data    T               `json:"data"`
}

// OKT returns a successful response with data.
func OKT[T any](data T) *APIResponse[T] {
	return &APIResponse[T]{Code: APIResponseCodeOK, Message: codeToMsg[APIResponseCodeOK], Data: data}
}

// ErrorT returns an error response with message and optional data.
func ErrorT[T any](code APIResponseCode, data T) *APIResponse[T] {
	return &APIResponse[T]{Code: code, Message: codeToMsg[code], Data: data}
}

package namespaceregistry

import "fmt"

// ErrorCode ist ein stabiler, maschinenlesbarer Fehlercode. Clients (z.B.
// ein Admin-GUI oder ein Resolver-Dienst) können anhand des Codes
// programmatisch reagieren, statt Freitext-Fehlermeldungen zu parsen.
type ErrorCode string

const (
	ErrInvalidArgument      ErrorCode = "INVALID_ARGUMENT"
	ErrInvalidNamespace     ErrorCode = "INVALID_NAMESPACE_FORMAT"
	ErrInvalidEndpoint      ErrorCode = "INVALID_RESOLVER_ENDPOINT"
	ErrNamespaceExists      ErrorCode = "NAMESPACE_ALREADY_EXISTS"
	ErrNamespaceNotFound    ErrorCode = "NAMESPACE_NOT_FOUND"
	ErrUnauthorized         ErrorCode = "UNAUTHORIZED_REGISTRAR"
	ErrIdentityUnavailable  ErrorCode = "CLIENT_IDENTITY_UNAVAILABLE"
	ErrLedgerRead           ErrorCode = "LEDGER_READ_FAILED"
	ErrLedgerWrite          ErrorCode = "LEDGER_WRITE_FAILED"
	ErrSerialization        ErrorCode = "SERIALIZATION_FAILED"
	ErrHistoryUnavailable   ErrorCode = "HISTORY_UNAVAILABLE"
)

// ContractError ist der einheitliche Fehlertyp, den alle öffentlichen
// Contract-Methoden zurückgeben. Er implementiert error und trägt
// zusätzlich einen stabilen Code sowie optionalen Kontext für Logs.
type ContractError struct {
	Code    ErrorCode
	Message string
	Cause   error
}

func (e *ContractError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *ContractError) Unwrap() error {
	return e.Cause
}

// newError erzeugt einen ContractError ohne zugrundeliegende Cause.
func newError(code ErrorCode, format string, args ...interface{}) *ContractError {
	return &ContractError{Code: code, Message: fmt.Sprintf(format, args...)}
}

// wrapError erzeugt einen ContractError mit einer zugrundeliegenden Cause
// (z.B. ein Fehler aus GetState/PutState/json.Marshal).
func wrapError(code ErrorCode, cause error, format string, args ...interface{}) *ContractError {
	return &ContractError{Code: code, Message: fmt.Sprintf(format, args...), Cause: cause}
}

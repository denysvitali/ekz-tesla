package ekz

import "errors"

type ErrorResponse struct {
	Message string `json:"message"`
}

var ErrTransactionNotFoundInTable = errors.New("transaction not found in table")

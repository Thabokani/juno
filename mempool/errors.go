package mempool

import "errors"

var (
	ErrMempoolEmpty = errors.New("mempool is empty")
	ErrFailedValidation = errors.New("transaction is not valid")
)
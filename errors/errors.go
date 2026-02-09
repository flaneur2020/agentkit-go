package errors

import (
	stderrors "errors"
	"io"
)

var ErrEOF = stderrors.New("agentkit: eof")

func IsEOF(err error) bool {
	if err == nil {
		return false
	}
	return stderrors.Is(err, ErrEOF) || stderrors.Is(err, io.EOF)
}

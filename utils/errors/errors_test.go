package errors

import (
	"testing"
	"log"
)

func TestNew(t *testing.T) {
	log.Print(New(ErrSuccess))
	log.Print(New(ErrUnknownError))
	log.Print(New(ErrUnstableNetwork))
	log.Print(New(ErrPermissionDeny))
	log.Print(New(ErrServiceUnderMaintaining))
	log.Print(New(ErrTooMuchRequest))
	log.Print(New(ErrServiceNotFound))

	log.Print(New(ErrNeedLogin))
	log.Print(New(ErrTokenExpired))
}

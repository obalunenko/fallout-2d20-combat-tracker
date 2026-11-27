package app

import (
	"fmt"

	"github.com/obalunenko/fallout/internal/domain"
)

// invalidf returns a formatted error marked domain.ErrInvalidInput.
func invalidf(format string, args ...any) error {
	return domain.InvalidInput(fmt.Errorf(format, args...))
}

// notFoundf returns a formatted error marked domain.ErrNotFound.
func notFoundf(format string, args ...any) error {
	return domain.NotFound(fmt.Errorf(format, args...))
}

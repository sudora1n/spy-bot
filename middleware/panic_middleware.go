package middleware

import (
	"fmt"
	"runtime/debug"

	"github.com/rs/zerolog/log"
)

func LogPanicHandler(recovered any) error {
	stack := debug.Stack()
	log.Error().
		Any("recovered", recovered).
		Str("stack", string(stack)).
		Msg("PANIC recovered")

	return fmt.Errorf("panic: %v", recovered)
}

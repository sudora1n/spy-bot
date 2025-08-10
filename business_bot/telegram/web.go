package telegram

import (
	"fmt"
	"net/http"
)

func RunTelegram(mux *http.ServeMux, port int) error {
	return http.ListenAndServe(fmt.Sprintf(":%d", port), mux)
}

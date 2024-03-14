package server_test

import (
	"net/http/httptest"
	"testing"

	"github.com/Harardin/rate-limit/internal/server"
)

func Test_Limiter(t *testing.T) {
	srv, _ := server.New(nil, nil)

	t.Run("loop request tests", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "localhost:20001/req", nil)
		for i := 0; i < 100; i++ {
			srv.HandleRequest(rr, req)
		}
	})
}

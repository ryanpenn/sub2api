package routes

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type runtimeStateStub struct {
	readyErr error
	draining bool
}

func (s runtimeStateStub) Ready(context.Context) error { return s.readyErr }
func (s runtimeStateStub) IsDraining() bool            { return s.draining }

func TestCommonRoutesKeepLivenessSeparateFromReadiness(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterCommonRoutes(router, runtimeStateStub{readyErr: errors.New("redis unavailable")})

	health := httptest.NewRecorder()
	router.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/health", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("/health status = %d, want %d", health.Code, http.StatusOK)
	}
	ready := httptest.NewRecorder()
	router.ServeHTTP(ready, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if ready.Code != http.StatusServiceUnavailable {
		t.Fatalf("/ready status = %d, want %d", ready.Code, http.StatusServiceUnavailable)
	}
}

func TestCommonRoutesReady(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterCommonRoutes(router, runtimeStateStub{})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("/ready status = %d, want %d", response.Code, http.StatusOK)
	}
}

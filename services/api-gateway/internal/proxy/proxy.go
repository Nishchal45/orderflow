package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/rs/zerolog"
)

// ServiceProxy forwards requests to a backend service.
type ServiceProxy struct {
	proxy  *httputil.ReverseProxy
	logger zerolog.Logger
}

// NewServiceProxy creates a reverse proxy to the given backend URL.
func NewServiceProxy(target string, logger zerolog.Logger) (*ServiceProxy, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(u)

	// Custom error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logger.Error().Err(err).Str("target", target).Str("path", r.URL.Path).Msg("proxy error")
		http.Error(w, `{"error":{"code":502,"message":"service unavailable"}}`, http.StatusBadGateway)
	}

	return &ServiceProxy{proxy: proxy, logger: logger}, nil
}

// ServeHTTP forwards the request to the backend service.
func (sp *ServiceProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sp.proxy.ServeHTTP(w, r)
}

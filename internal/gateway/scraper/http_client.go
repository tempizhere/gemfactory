// Package scraper реализует HTTP-клиент для парсинга релизов.
package scraper

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// HTTPClientConfig конфигурация HTTP клиента
type HTTPClientConfig struct {
	MaxIdleConns          int
	MaxIdleConnsPerHost   int
	IdleConnTimeout       time.Duration
	TLSHandshakeTimeout   time.Duration
	ResponseHeaderTimeout time.Duration
	DisableKeepAlives     bool
}

// NewHTTPClient создает новый HTTP клиент с оптимизированными настройками
func NewHTTPClient(config HTTPClientConfig, logger *zap.Logger) *http.Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          config.MaxIdleConns,
		MaxIdleConnsPerHost:   config.MaxIdleConnsPerHost,
		IdleConnTimeout:       config.IdleConnTimeout,
		TLSHandshakeTimeout:   config.TLSHandshakeTimeout,
		ResponseHeaderTimeout: config.ResponseHeaderTimeout,
		DisableKeepAlives:     config.DisableKeepAlives,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
			MinVersion:         tls.VersionTLS12,
		},
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
	}

	logger.Info("HTTP client created with connection pooling",
		zap.Int("max_idle_conns", config.MaxIdleConns),
		zap.Int("max_idle_conns_per_host", config.MaxIdleConnsPerHost),
		zap.Duration("idle_conn_timeout", config.IdleConnTimeout),
		zap.Duration("tls_handshake_timeout", config.TLSHandshakeTimeout),
		zap.Duration("response_header_timeout", config.ResponseHeaderTimeout),
		zap.Bool("disable_keep_alives", config.DisableKeepAlives))

	return client
}

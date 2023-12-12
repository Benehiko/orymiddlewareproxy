package oryproxy

import (
	"context"

	"github.com/rs/cors"
)

type DefaultConfigOptions func(*oryConfigDefault)

// Set the domain under which the proxy will set the cookie
func WithCookieDomain(domain string) DefaultConfigOptions {
	return func(oci *oryConfigDefault) {
		oci.cookieDomain = domain
	}
}

// Set a custom value for the proxy path prefix
// this is the path you are serving the proxy under
// e.g. /.ory/proxy
func WithPathPrefix(prefix string) DefaultConfigOptions {
	return func(oci *oryConfigDefault) {
		oci.proxyRoutePathPrefix = prefix
	}
}

// Sets the project API key that the proxy will use
// with social sign in requests
// this value can be omitted if using password or code login/registration
// flows.
func WithOryProjectAPIKey(apiKey string) DefaultConfigOptions {
	return func(oci *oryConfigDefault) {
		oci.oryProjectAPIKey = apiKey
	}
}

// Enable CORS
// use WithCorsOptions to configure CORS
// Only necessary if the request to the proxy is from
// a browser application running on a different origin
// than the proxy
func WithCorsEnabled(enabled bool) DefaultConfigOptions {
	return func(oci *oryConfigDefault) {
		oci.corsEnabled = enabled
	}
}

// Configure CORS
func WithCorsOptions(corsOptions *cors.Options) DefaultConfigOptions {
	return func(oci *oryConfigDefault) {
		oci.corsOptions = corsOptions
	}
}

// A function that is called when a request is proxied
// this is optional
func WithRequestLogger(logger RequestLogger) DefaultConfigOptions {
	return func(oci *oryConfigDefault) {
		oci.requestLogger = logger
	}
}

// a function that is called when a response is proxied
// this is optional
func WithResponseLogger(logger ResponseLogger) DefaultConfigOptions {
	return func(oci *oryConfigDefault) {
		oci.responseLogger = logger
	}
}

func NewDefaultConfig(oryProjectURL string, opts ...DefaultConfigOptions) OryConfig {
	oci := &oryConfigDefault{
		oryProjectURL:          oryProjectURL,
		proxyRoutePathPrefix:   "/.ory",
		cookieDomain:           "localhost",
		corsEnabled:            false,
		corsOptions:            &cors.Options{},
		trustXForwardedHeaders: false,
		requestLogger:          nil,
		responseLogger:         nil,
	}

	for _, opt := range opts {
		opt(oci)
	}

	return oci
}

var _ OryConfig = (*oryConfigDefault)(nil)

func (o *oryConfigDefault) CookieDomain(ctx context.Context) string {
	return o.cookieDomain
}

func (o *oryConfigDefault) ProxyRoutePathPrefix(ctx context.Context) string {
	return o.proxyRoutePathPrefix
}

func (o *oryConfigDefault) OryProjectURL(ctx context.Context) string {
	return o.oryProjectURL
}

func (o *oryConfigDefault) OryProjectAPIKey(ctx context.Context) string {
	return o.oryProjectAPIKey
}

func (o *oryConfigDefault) CorsEnabled(ctx context.Context) bool {
	return o.corsEnabled
}

func (o *oryConfigDefault) CorsOptions(ctx context.Context) *cors.Options {
	return o.corsOptions
}

func (o *oryConfigDefault) RequestLogger(ctx context.Context) RequestLogger {
	return o.requestLogger
}

func (o *oryConfigDefault) ResponseLogger(ctx context.Context) ResponseLogger {
	return o.responseLogger
}

func (o *oryConfigDefault) TrustXForwardedHeaders(ctx context.Context) bool {
	return o.trustXForwardedHeaders
}

package orymiddlewareproxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gorilla/mux"
	"github.com/ory/x/proxy"
	"github.com/rs/cors"
	"github.com/urfave/negroni"
)

type (
	middlewareContextKey string

	RequestLogger  func(ctx context.Context, req *httputil.ProxyRequest, body []byte)
	ResponseLogger func(ctx context.Context, resp *http.Response, body []byte)
	OryProxy       struct {
		config OryConfig
	}

	OryConfig interface {
		// RequestLogger is a function that is called when a request is proxied
		// to the upstream URL
		RequestLogger(context.Context) RequestLogger

		// ResponseLogger is a function that is called when a response is proxied
		// from the upstream URL
		ResponseLogger(context.Context) ResponseLogger

		// This is the cookie domain that the proxy will set on the response
		// so your project domain. This can be on localhost or something like
		// example.com
		CookieDomain(context.Context) string

		// ProxyRoutePathPrefix is the prefix under which the proxy is served.
		// for example, the proxy could be under /.ory/proxy, in which case the path prefix is /.ory/proxy
		ProxyRoutePathPrefix(context.Context) string

		// OryProjectURL is the URL of the Ory Project API
		// This is the URL that the proxy will forward requests to
		// the format is usually something like https://project-slug.projects.oryapis.com
		OryProjectURL(context.Context) string

		// OryProjectAPIKey is the API key that the proxy will use to authenticate with
		// the Ory Project API
		OryProjectAPIKey(context.Context) string

		// CorsEnabled is a flag to enable or disable CORS
		CorsEnabled(context.Context) bool

		// CorsOptions allows to configure CORS
		CorsOptions(context.Context) *cors.Options

		// TrustXForwardedHeaders is a flag that indicates whether the proxy should trust the
		// X-Forwarded-* headers or not.
		TrustXForwardedHeaders(context.Context) bool
	}

	oryConfigDefault struct {
		// requestLogger is a function that is called when a request is proxied
		// to the upstream URL
		requestLogger RequestLogger

		// responseLogger is a function that is called when a response is proxied
		// from the upstream URL
		responseLogger ResponseLogger

		// This is the cookie domain that the proxy will set on the response
		// so your project domain. This can be on localhost or something like
		// example.com
		cookieDomain string

		// OryProjectURL is the URL of the Ory Project API
		// This is the URL that the proxy will forward requests to
		// the format is usually something like https://project-slug.projects.oryapis.com
		oryProjectURL string

		// OryProjectAPIKey is the API key that the proxy will use to authenticate with
		// the Ory Project API
		oryProjectAPIKey string

		// PathPrefix is the prefix under which the proxy is served.
		// for example, the proxy could be under /.ory/proxy, in which case the path prefix is /.ory/proxy
		proxyRoutePathPrefix string

		// CorsEnabled is a flag to enable or disable CORS
		corsEnabled bool

		// CorsOptions allows to configure CORS
		corsOptions *cors.Options

		// trustXForwardedHeaders is a flag that indicates whether the proxy should trust the
		// X-Forwarded-* headers or not.
		trustXForwardedHeaders bool
	}
)

const (
	OriginalHostKey middlewareContextKey = "original-host"
)

func NewOryProxy(conf OryConfig) *OryProxy {
	return &OryProxy{
		config: conf,
	}
}

func (p *OryProxy) OryProxy() http.Handler {
	prefix := "/.ory"

	return proxy.New(
		func(ctx context.Context, r *http.Request) (context.Context, *proxy.HostConfig, error) {
			u, err := url.Parse(p.config.OryProjectURL(ctx))
			if err != nil {
				return ctx, nil, err
			}
			upstream := u.Host

			return ctx, &proxy.HostConfig{
				CookieDomain:          p.config.CookieDomain(ctx),
				UpstreamHost:          upstream,
				UpstreamScheme:        "https",
				TargetHost:            upstream,
				PathPrefix:            prefix,
				TrustForwardedHeaders: p.config.TrustXForwardedHeaders(ctx),
				TargetScheme:          "http",
				CorsEnabled:           p.config.CorsEnabled(ctx),
				CorsOptions:           p.config.CorsOptions(ctx),
			}, nil
		},
		proxy.WithReqMiddleware(func(r *httputil.ProxyRequest, c *proxy.HostConfig, body []byte) ([]byte, error) {
			if p.config.RequestLogger(r.In.Context()) != nil {
				p.config.RequestLogger(r.In.Context())(r.In.Context(), r, body)
			}

			// rewrite the request to point to the Middleware host instead of the Upstream host
			var host string
			if host = r.In.Header.Get("X-FORWARDED-HOST"); host == "" {
				host = r.In.Host
			}

			var proto string
			if proto = r.In.Header.Get("X-FORWARDED-PROTO"); proto == "" {
				if proto = r.In.URL.Scheme; proto == "" {
					proto = "http"
				}
			}

			rewriteHost := fmt.Sprintf("%s://%s%s", proto, host, c.PathPrefix)

			*r.Out = *r.Out.WithContext(context.WithValue(r.In.Context(), OriginalHostKey, rewriteHost))

			r.Out.Host = c.UpstreamHost
			r.Out.URL.Path = strings.TrimPrefix(r.Out.URL.Path, c.PathPrefix)
			r.Out.Header.Set("Ory-No-Custom-Domain-Redirect", "true")
			r.Out.Header.Set("Ory-Base-URL-Rewrite", rewriteHost)

			// used for social sign in on localhost
			if p.config.OryProjectAPIKey(r.In.Context()) != "" {
				r.Out.Header.Set("Ory-Base-URL-Rewrite-Token", p.config.OryProjectAPIKey(r.In.Context()))
			}

			r.Out.Header.Set("X-Forwarded-Host", host)
			r.Out.Header.Set("X-Forwarded-Proto", proto)

			return body, nil
		}),
		proxy.WithRespMiddleware(func(resp *http.Response, config *proxy.HostConfig, body []byte) ([]byte, error) {
			if p.config.ResponseLogger(resp.Request.Context()) != nil {
				p.config.ResponseLogger(resp.Request.Context())(resp.Request.Context(), resp, body)
			}

			originalHost, ok := resp.Request.Context().Value(OriginalHostKey).(string)
			if !ok {
				return nil, fmt.Errorf("could not get original host from context")
			}

			body = []byte(strings.ReplaceAll(string(body), config.UpstreamScheme+"://"+config.UpstreamHost, originalHost))

			if !strings.HasPrefix(resp.Header.Get("Location"), "https") {
				respCookies := resp.Cookies()
				resp.Header.Del("Set-Cookie")
				for _, c := range respCookies {
					rewriteCookie := &http.Cookie{
						Name:     c.Name,
						Value:    c.Value,
						Path:     c.Path,
						Domain:   c.Domain,
						Expires:  c.Expires,
						Secure:   c.Secure,
						HttpOnly: c.HttpOnly,
						SameSite: c.SameSite,
					}
					resp.Header.Add("Set-Cookie", rewriteCookie.String())
				}
			}

			return body, nil
		}),
	)
}

func (p *OryProxy) ListenAndServe(ctx context.Context, port int) error {
	r := mux.NewRouter()
	n := negroni.New()

	r.Handle(p.config.ProxyRoutePathPrefix(ctx), p.OryProxy())

	n.Use(negroni.NewRecovery())
	n.UseHandler(r)

	return http.ListenAndServe(fmt.Sprintf(":%d", port), p.OryProxy())
}

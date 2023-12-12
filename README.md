# Ory Proxy

A simple library that exposes a go `http.Handler` to proxy requests to and from
an Ory Network project. The library includes sane defaults and provides
an easy way to configure the proxy to fit your project needs.

The Ory CLI already provides proxy/tunnel capabilities, however,
when deploying a Go service to serverless infrastructure you 
need to add a custom domain to your Ory Network. 

This doesn't always work, since you might have test services or staging services
that do not need a custom domain setup. This library helps with that.


## Get Started

```sh
go get -u github.com/Benehiko/oryproxy
```

Setup the proxy as a standalone service

```go
package main

import "github.com/Benehiko/oryproxy"

func main() {
    config := oryproxy.NewDefaultConfig("https://project-slug.projects.oryapis.com")
    proxy := oryproxy.NewOryProxy(config)
    proxy.ListenAndServe(ctx, 3000)
}
```

Or integrate into your existing service

```go
package main

import (
    "github.com/Benehiko/oryproxy"
    "github.com/gorilla/mux"
)

func main() {
    config := oryproxy.NewDefaultConfig("https://project-slug.projects.oryapis.com")
    proxy := oryproxy.NewOryProxy(config)

    router := mux.NewRouter()
    // /.ory is the default path prefix for the proxy.
    // this can be changed.
    router.PathPrefix("/.ory").Handler(proxy.OryProxy())
}
```

## Limitations

Although this library proxies requests to and from Ory, it is still subject 
to rate-limiting on the Ory Network project subscription you have. Free-tier
projects can easily hit rate-limits and you should consider upgrading as soon
as you have real traffic on your service.

## Notice

This library is not officially supported by Ory. Please do not open support
tickets on the Ory repositories regarding this project nor ask for support
on their Slack channel.

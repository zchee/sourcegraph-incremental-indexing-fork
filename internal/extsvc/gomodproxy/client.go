package gomodproxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/inconshreveable/log15"
	"golang.org/x/mod/module"
	"golang.org/x/time/rate"

	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/httpcli"
	"github.com/sourcegraph/sourcegraph/internal/ratelimit"
	"github.com/sourcegraph/sourcegraph/lib/errors"
	"github.com/sourcegraph/sourcegraph/schema"
)

// A Client to Go module proxies.
type Client struct {
	urls    []string // list of proxy URLs
	cli     httpcli.Doer
	limiter *rate.Limiter
}

// NewClient returns a new Client for the given configuration.
func NewClient(config *schema.GoModulesConnection, cli httpcli.Doer) *Client {
	var requestsPerHour float64
	if config.RateLimit == nil || !config.RateLimit.Enabled {
		requestsPerHour = math.Inf(1)
	} else {
		requestsPerHour = config.RateLimit.RequestsPerHour
	}

	return &Client{
		urls:    config.Urls,
		cli:     cli,
		limiter: rate.NewLimiter(rate.Limit(requestsPerHour/3600.0), 100),
	}
}

// GetVersion gets a single version of the given module if it exists.
func (c *Client) GetVersion(ctx context.Context, mod, version string) (*module.Version, error) {
	escapedVersion, err := module.EscapeVersion(version)
	if err != nil {
		return nil, errors.Wrap(err, "failed to escape version")
	}

	respBody, err := c.get(ctx, mod, "@v", escapedVersion+".info")
	if err != nil {
		return nil, err
	}

	var v struct{ Version string }
	if err = json.Unmarshal(respBody, &v); err != nil {
		return nil, err
	}

	return &module.Version{Path: mod, Version: v.Version}, nil
}

// ListVersions list all versions of the given module.
func (c *Client) ListVersions(ctx context.Context, mod string) (vs []module.Version, err error) {
	respBody, err := c.get(ctx, mod, "@v/list")
	if err != nil {
		return nil, err
	}

	for _, version := range bytes.Split(respBody, []byte("\n")) {
		vs = append(vs, module.Version{Path: mod, Version: string(version)})
	}

	return vs, nil
}

// GetZip returns the zip archive bytes of the given module and version.
func (c *Client) GetZip(ctx context.Context, mod, version string) ([]byte, error) {
	escapedVersion, err := module.EscapeVersion(version)
	if err != nil {
		return nil, errors.Wrap(err, "failed to escape version")
	}

	return c.get(ctx, mod, "@v", escapedVersion+".zip")
}

func (c *Client) get(ctx context.Context, mod string, paths ...string) (respBody []byte, err error) {
	escapedMod, err := module.EscapePath(mod)
	if err != nil {
		return nil, errors.Wrap(err, "failed to escape module path")
	}

	for _, baseURL := range c.urls {
		limiter := ratelimit.DefaultRegistry.GetOrSet(baseURL, c.limiter)

		startWait := time.Now()
		if err := limiter.Wait(ctx); err != nil {
			return nil, err
		}

		if d := time.Since(startWait); d > 200*time.Millisecond {
			log15.Warn("go modules proxy client self-enforced API rate limit: request delayed longer than expected due to rate limit", "delay", d)
		}

		reqURL, err := url.Parse(baseURL)
		if err != nil {
			return nil, errors.Errorf("invalid go modules proxy URL %q", baseURL)
		}
		reqURL.Path = path.Join(escapedMod, path.Join(paths...))

		req, err := http.NewRequestWithContext(ctx, "GET", reqURL.String(), nil)
		if err != nil {
			return nil, err
		}

		respBody, err = c.do(req)
		if err == nil || !errcode.IsNotFound(err) {
			break
		}
	}

	return respBody, err
}

func (c *Client) do(req *http.Request) ([]byte, error) {
	resp, err := c.cli.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	bs, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// https://go.dev/ref/mod#goproxy-protocol
	// Successful HTTP responses must have the status code 200 (OK).
	// Redirects (3xx) are followed. Responses with status codes 4xx and 5xx are treated as errors.
	// The error codes 404 (Not Found) and 410 (Gone) indicate that the requested module or version is not available
	// on the proxy, but it may be found elsewhere.
	// Error responses should have content type text/plain with charset either utf-8 or us-ascii.

	if resp.StatusCode != http.StatusOK {
		return nil, &Error{Path: req.URL.Path, Code: resp.StatusCode, Message: string(bs)}
	}

	return bs, nil
}

// Error returned from an HTTP request to a Go module proxy.
type Error struct {
	Path    string
	Code    int
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("bad go module proxy response with status code %d for %s: %s", e.Code, e.Path, e.Message)
}

func (e *Error) IsNotFound() bool {
	return e.Code == http.StatusNotFound || e.Code == http.StatusGone
}
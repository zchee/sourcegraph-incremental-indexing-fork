package executorqueue

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
)

func TestReverseProxySimple(t *testing.T) {
	originServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	defer originServer.Close()

	url, err := url.Parse(originServer.URL)
	if err != nil {
		t.Fatalf("unexpected error parsing url: %s", err)
	}

	proxyServer := httptest.NewServer(reverseProxy(url))
	defer proxyServer.Close()

	req, err := http.NewRequest("GET", proxyServer.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error creating request: %s", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("unexpected error performing request: %s", err)
	}
	if resp.StatusCode != http.StatusTeapot {
		t.Errorf("unexpected status code. want=%d have=%d", http.StatusTeapot, resp.StatusCode)
	}
}

func TestReverseProxyTargetPath(t *testing.T) {
	oldGetRest := getRest
	getRest = func(r *http.Request) string { return "/bar/baz" }
	defer func() { getRest = oldGetRest }()

	originServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/foo/bar/baz" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusTeapot)
	}))
	defer originServer.Close()

	url, err := url.Parse(originServer.URL + "/foo")
	if err != nil {
		t.Fatalf("unexpected error parsing url: %s", err)
	}

	proxyServer := httptest.NewServer(reverseProxy(url))
	defer proxyServer.Close()

	req, err := http.NewRequest("GET", proxyServer.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error creating request: %s", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("unexpected error performing request: %s", err)
	}
	if resp.StatusCode != http.StatusTeapot {
		t.Errorf("unexpected status code. want=%d have=%d", http.StatusTeapot, resp.StatusCode)
	}
}

func TestReverseProxyHeaders(t *testing.T) {
	originServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("baz", r.Header.Get("foo"))
		w.WriteHeader(http.StatusTeapot)
	}))
	defer originServer.Close()

	url, err := url.Parse(originServer.URL)
	if err != nil {
		t.Fatalf("unexpected error parsing url: %s", err)
	}

	proxyServer := httptest.NewServer(reverseProxy(url))
	defer proxyServer.Close()

	req, err := http.NewRequest("GET", proxyServer.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error creating request: %s", err)
	}
	req.Header.Add("foo", "bar")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("unexpected error performing request: %s", err)
	}
	if resp.StatusCode != http.StatusTeapot {
		t.Errorf("unexpected status code. want=%d have=%d", http.StatusTeapot, resp.StatusCode)
	}
	if value := resp.Header.Get("baz"); value != "bar" {
		t.Errorf("unexpected header value. want=%s have=%s", "bar", value)
	}
}

func TestReverseProxyRedirectWithPayload(t *testing.T) {
	var originServer *httptest.Server
	originServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/foo/foo/foo" {
			http.Redirect(w, r, originServer.URL+filepath.Join(r.URL.Path, "foo"), http.StatusTemporaryRedirect)
			return
		}

		contents, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Add("payload", string(contents))
		w.WriteHeader(http.StatusTeapot)
	}))
	defer originServer.Close()

	url, err := url.Parse(originServer.URL)
	if err != nil {
		t.Fatalf("unexpected error parsing url: %s", err)
	}

	proxyServer := httptest.NewServer(reverseProxy(url))
	defer proxyServer.Close()

	req, err := http.NewRequest("POST", proxyServer.URL, bytes.NewReader([]byte("foobarbaz")))
	if err != nil {
		t.Fatalf("unexpected error creating request: %s", err)
	}
	req.Header.Add("foo", "bar")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("unexpected error performing request: %s", err)
	}
	if resp.StatusCode != http.StatusTeapot {
		t.Errorf("unexpected status code. want=%d have=%d", http.StatusTeapot, resp.StatusCode)
	}
	if value := resp.Header.Get("payload"); value != "foobarbaz" {
		t.Errorf("unexpected header value. want=%s have=%s", "foobarbaz", value)
	}
}

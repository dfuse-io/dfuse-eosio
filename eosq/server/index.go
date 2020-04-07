package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"mime"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/koding/websocketproxy"
	"go.uber.org/zap"
)

var staticPathRegexp = regexp.MustCompile("^/static/")

type EOSQueryUI struct {
	hasGzipFileMap *sync.Map
	reverseProxy   *httputil.ReverseProxy
	websocketProxy *websocketproxy.WebsocketProxy
}

func NewDevServerProxy() *EOSQueryUI {
	host := "localhost:3000"
	if newHost := os.Getenv("DEV_HOST"); newHost != "" {
		host = newHost
	}

	u, err := url.Parse(fmt.Sprintf("http://%s/", host))
	if err != nil {
		zlog.Fatal("parsing url", zap.Error(err), zap.String("host", host))
	}
	reverseProxy := httputil.NewSingleHostReverseProxy(u)
	reverseProxy.Transport = indexTemplater{}

	wsURL, _ := url.Parse(fmt.Sprintf("ws://%s/", host))

	wsProxy := websocketproxy.NewProxy(wsURL)

	return &EOSQueryUI{
		hasGzipFileMap: new(sync.Map),
		reverseProxy:   reverseProxy,
		websocketProxy: wsProxy,
	}
}

func (ui *EOSQueryUI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch *deploymentEnv {
	case "staging", "prod":
		ui.ServeHTTPForProduction(w, r)
	default:
		ui.ServeHTTPForDevelopment(w, r)
	}
}

func (ui *EOSQueryUI) ServeHTTPForProduction(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Forwarded-Proto") == "http" {
		target := "https://" + r.Host + r.URL.Path
		if len(r.URL.RawQuery) > 0 {
			target += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, target, http.StatusMovedPermanently)
		return
	} else {
		w.Header().Add("Strict-Transport-Security", "max-age=600; includeSubDomains; preload")
	}

	defaultSource := *htmlRoot

	cwd, _ := os.Getwd()

	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}

	if path == "/robots.txt" {
		if err := serveRobotsTxt(w, r); err != nil {
			zlog.Error("serve robots txt", zap.Error(err))
			http.Error(w, "error rendering robots.txt", 500)
		}
		return
	}

	filePath := filepath.Join(cwd, defaultSource, path)
	var fileFound bool
	_, err := os.Stat(filePath)
	if err == nil {
		fileFound = true
	}

	if fileFound && path == "/service-worker.js" || path == "/custom-sw.js" {
		bustCache(w)
		http.FileServer(http.Dir(defaultSource)).ServeHTTP(w, r)
	} else if fileFound && path != "/index.html" {
		if acceptsGzip(r) {
			if _, ok := ui.hasGzipFileMap.Load(path); ok {
				serveCompressedFile(w, r, defaultSource+path)
			} else if fileExists(defaultSource + path + ".gz") {
				ui.hasGzipFileMap.Store(path, true)
				serveCompressedFile(w, r, defaultSource+path)
			} else {
				http.FileServer(http.Dir(defaultSource)).ServeHTTP(w, r)
			}
		} else {
			http.FileServer(http.Dir(defaultSource)).ServeHTTP(w, r)
		}
	} else if staticPathRegexp.MatchString(path) {
		http.Error(w, "resource not found", 404)
	} else {
		bustCache(w)
		if err := serveIndexHTML(w, r, defaultSource+"/index.html"); err != nil {
			zlog.Error("serve index html", zap.Error(err))
			http.Error(w, "error rendering index", 500)
		}
	}
}

func (ui *EOSQueryUI) ServeHTTPForDevelopment(w http.ResponseWriter, r *http.Request) {
	r.Header.Del("Accept-Encoding")

	if r.Header.Get("Connection") == "Upgrade" {
		zlog.Debug("proxying websocket connection (upgrade)", zap.String("path", r.URL.Path))
		ui.websocketProxy.ServeHTTP(w, r)
	} else {
		zlog.Debug("proxying", zap.String("path", r.URL.Path))
		ui.reverseProxy.ServeHTTP(w, r)
	}
}

// indexTemplater will template-in the theme variables, on-the-fly,
// from the NodeJS dev server.
type indexTemplater struct{}

func bustCache(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
}

func (it indexTemplater) RoundTrip(r *http.Request) (resp *http.Response, err error) {
	resp, err = http.DefaultTransport.RoundTrip(r)
	if err != nil {
		return
	}

	if resp.ContentLength < 3000 && resp.Header.Get("Content-Type") == "text/html; charset=UTF-8" {
		cnt, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return resp, err
		}
		_ = resp.Body.Close()

		reader, err := templatedIndex(cnt)
		if err != nil {
			return resp, err
		}

		resp.Header.Set("Cache-Control", "no-cache, no-store, must-revalidate")
		resp.Header.Set("Pragma", "no-cache")
		resp.Header.Set("Expires", "0")
		resp.Header.Set("Content-Length", fmt.Sprintf("%d", reader.Size()))
		resp.Body = ioutil.NopCloser(reader)
	}

	return
}

func serveRobotsTxt(w http.ResponseWriter, r *http.Request) error {
	content := "User-agent: *\nDisallow: /\n"
	if *deploymentEnv == "prod" {
		content = "User-agent: *\nnDisallow:\n"
	}

	_, err := w.Write([]byte(content))
	return err
}

func serveCompressedFile(w http.ResponseWriter, r *http.Request, filePath string) {
	w.Header().Set(contentEncoding, "gzip")
	w.Header().Set(contentType, mime.TypeByExtension(filepath.Ext(filePath)))
	w.Header().Del(contentLength)

	http.ServeFile(w, r, filePath+".gz")
}

func serveIndexHTML(w http.ResponseWriter, r *http.Request, indexFileName string) error {
	mtime, err := os.Stat(indexFileName)
	if err != nil {
		return err
	}

	cnt, err := ioutil.ReadFile(indexFileName)
	if err != nil {
		return err
	}

	reader, err := templatedIndex(cnt)
	if err != nil {
		return err
	}

	http.ServeContent(w, r, "index.html", mtime.ModTime(), reader)

	return nil
}

func templatedIndex(content []byte) (*bytes.Reader, error) {
	cnt, err := ioutil.ReadFile(*frontendConfigPath)
	if err != nil {
		return nil, err
	}

	config := map[string]interface{}{}
	err = json.Unmarshal(cnt, &config)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal config: %s", err)
	}

	tpl, err := template.New("index.html").Funcs(template.FuncMap{
		"json": func(v interface{}) (template.JS, error) {
			cnt, err := json.Marshal(v)
			return template.JS(cnt), err
		},
	}).Delims("--==", "==--").Parse(string(content))
	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	if err := tpl.Execute(buf, config); err != nil {
		return nil, err
	}

	return bytes.NewReader(buf.Bytes()), nil
}

func fileExists(f string) bool {
	_, err := os.Stat(f)
	if os.IsNotExist(err) {
		return false
	}

	return err == nil
}

// Extracted from https://github.com/NYTimes/gziphandler
//  * acceptsGzip - https://github.com/NYTimes/gziphandler/blob/master/gzip.go#L450
//  * parseEncodings, parseCoding  - https://github.com/NYTimes/gziphandler/blob/master/gzip.go#L476

type codings map[string]float64

const (
	acceptEncoding  = "Accept-Encoding"
	contentEncoding = "Content-Encoding"
	contentType     = "Content-Type"
	contentLength   = "Content-Length"
)

const (
	// DefaultQValue is the default qvalue to assign to an encoding if no explicit qvalue is set.
	// This is actually kind of ambiguous in RFC 2616, so hopefully it's correct.
	// The examples seem to indicate that it is.
	DefaultQValue = 1.0
)

func acceptsGzip(r *http.Request) bool {
	acceptedEncodings, _ := parseEncodings(r.Header.Get(acceptEncoding))
	return acceptedEncodings["gzip"] > 0.0
}

// parseEncodings attempts to parse a list of codings, per RFC 2616, as might
// appear in an Accept-Encoding header. It returns a map of content-codings to
// quality values, and an error containing the errors encountered. It's probably
// safe to ignore those, because silently ignoring errors is how the internet
// works.
//
// See: http://tools.ietf.org/html/rfc2616#section-14.3.
func parseEncodings(s string) (codings, error) {
	c := make(codings)
	var e []string

	for _, ss := range strings.Split(s, ",") {
		coding, qvalue, err := parseCoding(ss)

		if err != nil {
			e = append(e, err.Error())
		} else {
			c[coding] = qvalue
		}
	}

	// TODO (adammck): Use a proper multi-error struct, so the individual errors
	//                 can be extracted if anyone cares.
	if len(e) > 0 {
		return c, fmt.Errorf("errors while parsing encodings: %s", strings.Join(e, ", "))
	}

	return c, nil
}

// parseCoding parses a single conding (content-coding with an optional qvalue),
// as might appear in an Accept-Encoding header. It attempts to forgive minor
// formatting errors.
func parseCoding(s string) (coding string, qvalue float64, err error) {
	for n, part := range strings.Split(s, ";") {
		part = strings.TrimSpace(part)
		qvalue = DefaultQValue

		if n == 0 {
			coding = strings.ToLower(part)
		} else if strings.HasPrefix(part, "q=") {
			qvalue, err = strconv.ParseFloat(strings.TrimPrefix(part, "q="), 64)

			if qvalue < 0.0 {
				qvalue = 0.0
			} else if qvalue > 1.0 {
				qvalue = 1.0
			}
		}
	}

	if coding == "" {
		err = fmt.Errorf("empty content-coding")
	}

	return
}

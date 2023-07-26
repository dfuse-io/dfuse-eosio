package apiproxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/streamingfast/shutter"
	"go.uber.org/zap"
	"golang.org/x/crypto/acme/autocert"
)

type proxy struct {
	*shutter.Shutter
	config        *Config
	httpServer    *http.Server
	httpsServer   *http.Server
	dgraphqlProxy *httputil.ReverseProxy
	eoswsProxy    *httputil.ReverseProxy
	nodeosProxy   *httputil.ReverseProxy
	rootProxy     *httputil.ReverseProxy
}

func newProxy(config *Config) *proxy {
	createProxy := func(addr string) *httputil.ReverseProxy {
		return httputil.NewSingleHostReverseProxy(&url.URL{Host: "localhost" + addr, Scheme: "http"})
	}

	return &proxy{
		Shutter:       shutter.New(),
		config:        config,
		dgraphqlProxy: createProxy(config.DgraphqlHTTPAddr),
		eoswsProxy:    createProxy(config.EoswsHTTPAddr),
		nodeosProxy:   createProxy(config.NodeosHTTPAddr),
		rootProxy:     createProxy(config.RootHTTPAddr),
	}
}

func (p *proxy) Launch() error {
	zlog.Info("starting dashboard server")
	p.OnTerminating(p.cleanUp)

	router := mux.NewRouter()

	originsOptions := handlers.AllowedOrigins([]string{"*"})
	headersOptions := handlers.AllowedHeaders([]string{"authorization"})

	router.Methods("OPTIONS").PathPrefix("/").Handler(handlers.CORS(originsOptions, headersOptions)(router))

	// FIXME: This is most probably a TCP proxy to GRPC server, how to handle that?
	// "/dfuse.eosio.v1.GraphQL" dgraphqlProxy
	// "/grpc.reflection.v1alpha.ServerReflection" dgraphqlProxy

	router.PathPrefix("/graphql").Handler(p.dgraphqlProxy)
	router.PathPrefix("/graphiql").Handler(p.dgraphqlProxy)
	router.PathPrefix("/v1/chain/push_transaction").Handler(p.eoswsProxy)
	router.PathPrefix("/v1/chain/send_transaction").Handler(p.eoswsProxy)
	router.PathPrefix("/v1/chain").Handler(p.nodeosProxy)
	router.PathPrefix("/v1/stream").Handler(p.eoswsProxy)
	router.PathPrefix("/v1").Handler(p.eoswsProxy)
	router.PathPrefix("/v0").Handler(p.eoswsProxy)
	router.PathPrefix("/").Handler(p.rootProxy)

	errorLogger, err := zap.NewStdLogAt(zlog, zap.ErrorLevel)
	if err != nil {
		return fmt.Errorf("unable to create error logger: %w", err)
	}

	p.httpServer = &http.Server{
		Addr:     p.config.HTTPListenAddr,
		Handler:  router,
		ErrorLog: errorLogger,
	}

	zlog.Info("starting http server", zap.String("listen_addr", p.config.HTTPListenAddr))

	if p.config.HTTPSListenAddr != "" {
		zlog.Info("starting SSL listener", zap.Any("domains", p.config.AutocertDomains))
		m := &autocert.Manager{
			Cache:      autocert.DirCache(p.config.AutocertCacheDir),
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(p.config.AutocertDomains...),
		}
		p.httpsServer = &http.Server{
			Addr:      p.config.HTTPSListenAddr,
			TLSConfig: m.TLSConfig(),
			Handler:   router,
			ErrorLog:  errorLogger,
		}
		go func() {
			err := p.httpsServer.ListenAndServeTLS("", "")
			if err != nil {
				p.Shutdown(err)
			}
		}()
	}

	return p.httpServer.ListenAndServe()
}

func (p *proxy) cleanUp(err error) {
	if p.httpServer != nil {
		p.httpServer.Close()
	}
}

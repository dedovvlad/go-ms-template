// This file is safe to edit. Once it exists it will not be overwritten

package restapi

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/vrischmann/envconfig"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"test_tmp/internal/generated/restapi/operations"
	"test_tmp/internal/generated/restapi/operations/test"

	"github.com/dedovvlad/go-ms-template/internal/app"
	"github.com/dedovvlad/go-ms-template/internal/generated/restapi/healthcheck"
)

//go:generate swagger generate server --target ../../generated --name GoMsTemplate --spec ../../../swagger-doc/swagger.yml --template-dir ./swagger-gen/templates --principal interface{}

func configureFlags(api *operations.GoMsTemplateAPI) {
	// api.CommandLineOptionsGroups = []swag.CommandLineOptionsGroup{ ... }
}

func configureAPI(api *operations.GoMsTemplateAPI) http.Handler {
	// configure the api here
	api.ServeError = errors.ServeError
	metrics := prometheus.NewRegistry()
	initMetrics(metrics)
	srv := app.New(metrics)

	api.ServerShutdown = srv.OnShutdown
	type servMiddleware interface {
		Middleware(next http.Handler) http.Handler
	}
	if impl, ok := (interface{}(srv)).(servMiddleware); ok {
		api.Middleware = func(builder middleware.Builder) http.Handler {
			apiHandler := api.Context().RoutesHandler(builder)
			return impl.Middleware(apiHandler)
		}
	}

	// Set your custom logger if needed. Default one is log.Printf
	// Expected interface func(string, ...interface{})
	//
	// Example:
	// api.Logger = log.Printf

	api.UseSwaggerUI()
	// To continue using redoc as your UI, uncomment the following line
	// api.UseRedoc()

	api.JSONConsumer = runtime.JSONConsumer()

	api.JSONProducer = runtime.JSONProducer()

	if api.TestTestHandler == nil {
		api.TestTestHandler = test.TestHandlerFunc(func(params test.TestParams) middleware.Responder {
			return middleware.NotImplemented("operation test.Test has not yet been implemented")
		})
	}

	api.PreServerShutdown = func() {}

	srv.SetupSwaggerHandlers(api)
	if err := srv.ConfigureService(); err != nil {
		panic(err)
	}
	return setupCustomRoutes(
		srv,
		setupGlobalMiddleware(
			metricsMiddleware(
				api.Context(),
				recoverMiddleware(
					tracerMiddleware(api.Context(), api.Serve(setupMiddlewares)),
				),
			),
		),
	)
}

type ResponseWriterWithHTTPCode struct {
	wrapped http.ResponseWriter
	Code    int
}

func NewResponseWriterWithHTTPCode(wrapped http.ResponseWriter) *ResponseWriterWithHTTPCode {
	return &ResponseWriterWithHTTPCode{wrapped: wrapped}
}

func (rw *ResponseWriterWithHTTPCode) Header() http.Header {
	return rw.wrapped.Header()
}

func (rw *ResponseWriterWithHTTPCode) Write(content []byte) (int, error) {
	return rw.wrapped.Write(content)
}

func (rw *ResponseWriterWithHTTPCode) WriteHeader(statusCode int) {
	rw.Code = statusCode
	rw.wrapped.WriteHeader(statusCode)
}

func metricsMiddleware(ctx *middleware.Context, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// собираем данные об http вызове - время выполнения, хендлер, код ответа и метод запроса
		ts := time.Now()
		var handlerName string
		respWriter := NewResponseWriterWithHTTPCode(w)
		route, _, exists := ctx.RouteInfo(r)

		defer func() {
			if exists && route != nil && route.Operation != nil {
				handlerName = route.Operation.ID
			} else {
				handlerName = formatHandlerName(r)
			}

			statReqDurations.With(prometheus.Labels{"handler": handlerName}).Observe(time.Since(ts).Seconds())
			statReqCount.With(prometheus.Labels{
				"code":    strconv.Itoa(respWriter.Code),
				"method":  r.Method,
				"handler": handlerName,
			}).Inc()
		}()

		next.ServeHTTP(respWriter, r)
	})
}

func formatHandlerName(r *http.Request) string {
	parts := strings.Split(r.RequestURI, "/")
	for i, p := range parts {
		if len(p) < 6 {
			continue
		}
		for j, ch := range p {
			if ch >= '0' && ch <= '9' {
				parts[i] = "{param}"
				break
			}
			if ch == '?' {
				parts[i] = p[:j]
				break
			}
		}
	}

	return r.Method + " " + strings.Join(parts, "/")
}

// setupCustomRoutes creates http handler to serve custom routes
func setupCustomRoutes(srv *app.Service, next http.Handler) http.Handler {
	host, _ := os.Hostname()
	hCheck := &healthcheck.Healthcheck{
		AppName:    version.SERVICE_NAME,
		Version:    version.VERSION,
		ServerName: host,
	}
	for _, checkerFunc := range srv.HealthCheckers() {
		hCheck.AddChecker(checkerFunc)
	}

	mux := http.NewServeMux()
	mux.Handle("/", next)
	mux.Handle("/health/check", healthcheck.Handler(hCheck))
	mux.Handle("/metrics", promhttp.HandlerFor(srv.Metrics(), promhttp.HandlerOpts{}))

	cfg := struct{ PprofEnabled bool }{}
	envconfig.InitWithPrefix(&cfg, version.SERVICE_NAME)

	if cfg.PprofEnabled {
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}

	mux.HandleFunc("/version", versionHandler)
	startTime = time.Now()
	return mux
}

var startTime time.Time

const infoTpl = `{"app": %q, "version": %q, "buildTime": %q, "upTime": %.6q}`

func versionHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, infoTpl, version.SERVICE_NAME, version.VERSION, version.BUILD_TIME, time.Since(startTime))
}

var (
	statReqCount     *prometheus.CounterVec
	statReqDurations *prometheus.HistogramVec
)

func initMetrics(metrics *prometheus.Registry) {
	metrics.MustRegister(prometheus.NewGoCollector())
	metrics.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{
		PidFn:     nil,
		Namespace: version.SERVICE_NAME,
	}))

	statReqCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: version.SERVICE_NAME,
		Name:      "rest_requests_total",
		Help:      "Total number of rest requests",
	}, []string{"method", "code", "handler"})

	statReqDurations = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: version.SERVICE_NAME,
		Name:      "rest_request_duration_seconds",
		Help:      "Rest request duration",
		Buckets:   []float64{0.005, 0.01, 0.05, 0.1, 0.5, 1, 5},
	}, []string{"handler"})

	metrics.MustRegister(statReqCount, statReqDurations)
}

func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				sentry.CaptureException(fmt.Errorf("%v", r))
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, req)
	})
}

func tracerMiddleware(mctx *middleware.Context, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sctx, err := tracer.Extract(tracer.HTTPHeadersCarrier(r.Header))
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		var handlerName string
		route, _, exists := mctx.RouteInfo(r)
		if exists && route != nil && route.Operation != nil {
			handlerName = route.Operation.ID
		} else {
			handlerName = fmt.Sprintf("%s %s", r.Method, r.URL.String())
		}
		span := tracer.StartSpan(handlerName, tracer.ChildOf(sctx))
		defer span.Finish()

		ctx := tracer.ContextWithSpan(r.Context(), span)
		// mesh
		ctx = mapServiceMeshHeadersToContext(ctx, r.Header)
		r = r.Clone(ctx)

		next.ServeHTTP(w, r)
	})
}

const (
	serviceMeshHeaderPrefix = "X-Service-"
)

func mapServiceMeshHeadersToContext(ctx context.Context, headers http.Header) context.Context {
	headersMesh := make(map[string]string)
	for name := range headers {
		if strings.HasPrefix(name, serviceMeshHeaderPrefix) {
			headersMesh[strings.ToLower(name[len(serviceMeshHeaderPrefix):])] = headers.Get(name)
		}
	}
	ctx = context.WithValue(ctx, "mesh", headersMesh)

	return ctx
}

// The TLS configuration before HTTPS server starts.
func configureTLS(tlsConfig *tls.Config) {
	// Make all necessary changes to the TLS configuration here.
}

// As soon as server is initialized but not run yet, this function will be called.
// If you need to modify a config, store server instance to stop it individually later, this is the place.
// This function can be called multiple times, depending on the number of serving schemes.
// scheme value will be set accordingly: "http", "https" or "unix".
func configureServer(s *http.Server, scheme, addr string) {
}

// The middleware configuration is for the handler executors. These do not apply to the swagger.json document.
// The middleware executes after routing but before authentication, binding and validation.
func setupMiddlewares(handler http.Handler) http.Handler {
	return handler
}

// The middleware configuration happens before anything, this middleware also applies to serving the swagger.json document.
// So this is a good place to plug in a panic handling middleware, logging and metrics.
func setupGlobalMiddleware(handler http.Handler) http.Handler {
	return handler
}

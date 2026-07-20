package support_http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/labstack/echo/v5"
)

type bodyDumpResponseWriter struct {
	http.ResponseWriter
	body   *bytes.Buffer
	status int
}

func (w *bodyDumpResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *bodyDumpResponseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *bodyDumpResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// LoggingMiddleware logs every request. It fulfills:
// T004: Info log for normal requests.
// T002: Warn log for 4xx validation errors with failure details.
func LoggingMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			req := c.Request()
			start := time.Now()

			resBody := new(bytes.Buffer)
			origWriter := c.Response()
			wrappedWriter := &bodyDumpResponseWriter{ResponseWriter: origWriter, body: resBody}
			c.SetResponse(wrappedWriter)

			err := next(c)

			latency := time.Since(start)
			logger := logging.Default()
			ctx := req.Context()

			status := wrappedWriter.status
			if status == 0 {
				status = http.StatusOK
			}
			if err != nil {
				// If an error is returned, the error handler will eventually write the response.
				// We can infer the status code.
				status = echo.StatusCode(err)
				if status == 0 {
					status = http.StatusInternalServerError
				}
			}

			switch {
			case status >= 500:
				logger.Info(ctx, "request completed",
					"method", req.Method,
					"path", req.URL.Path,
					"status", status,
					"latency_ms", latency.Milliseconds(),
				)
			case status >= 400:
				reason := ""
				if resBody.Len() > 0 {
					var errResp struct {
						Message string `json:"message"`
						Error   string `json:"error"`
					}
					if jsonErr := json.Unmarshal(resBody.Bytes(), &errResp); jsonErr == nil {
						reason = errResp.Message
						if reason == "" {
							reason = errResp.Error
						}
					}
				}
				if reason == "" && err != nil {
					reason = err.Error()
				}
				if status == 400 {
					logger.Warn(ctx, "client request validation failed",
						"method", req.Method,
						"path", req.URL.Path,
						"status", status,
						"latency_ms", latency.Milliseconds(),
						"reason", reason,
					)
				} else {
					logger.Warn(ctx, "client request error",
						"method", req.Method,
						"path", req.URL.Path,
						"status", status,
						"latency_ms", latency.Milliseconds(),
						"reason", reason,
					)
				}
			default:
				logger.Info(ctx, "request completed",
					"method", req.Method,
					"path", req.URL.Path,
					"status", status,
					"latency_ms", latency.Milliseconds(),
				)
			}

			return err
		}
	}
}

package handlers

import (
	"fmt"
	"log"
	"github.com/openfaas/faas/gateway/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"strconv"
	"strings"
	"time"
)

// HTTPNotifier notify about HTTP request/response
type HTTPNotifier interface {
	Notify(method string, URL string, originalURL string, statusCode int, duration time.Duration)
}

// PrometheusServiceNotifier notifier for core service endpoints
type PrometheusServiceNotifier struct {
	ServiceMetrics *metrics.ServiceMetricOptions
}

// Notify about service metrics
func (psn PrometheusServiceNotifier) Notify(method string, URL string, originalURL string, statusCode int, duration time.Duration) {
	code := fmt.Sprintf("%d", statusCode)
	path := urlToLabel(URL)

	psn.ServiceMetrics.Counter.WithLabelValues(method, path, code).Inc()
	psn.ServiceMetrics.Histogram.WithLabelValues(method, path, code).Observe(duration.Seconds())
}

func urlToLabel(path string) string {
	if len(path) > 0 {
		path = strings.TrimRight(path, "/")
	}
	if path == "" {
		path = "/"
	}
	return path
}

// PrometheusFunctionNotifier records metrics to Prometheus
type PrometheusFunctionNotifier struct {
	Metrics *metrics.MetricOptions
}

// Notify records metrics in Prometheus
func (p PrometheusFunctionNotifier) Notify(method string, URL string, originalURL string, statusCode int, duration time.Duration) {
	seconds := duration.Seconds()
	serviceName := getServiceName(originalURL)

	p.Metrics.GatewayFunctionsHistogram.
		WithLabelValues(serviceName).
		Observe(seconds)

	code := strconv.Itoa(statusCode)

	p.Metrics.GatewayFunctionInvocation.
		With(prometheus.Labels{"function_name": serviceName, "code": code}).
		Inc()
}

func getServiceName(urlValue string) string {
	var serviceName string
	forward := "/function/"
	forwardChain := "/chain/"
	if strings.HasPrefix(urlValue, forward) || strings.HasPrefix(urlValue, forwardChain){
		// With a path like `/function/xyz/rest/of/path?q=a`, the service
		// name we wish to locate is just the `xyz` portion.  With a positive
		// match on the regex below, it will return a three-element slice.
		// The item at index `0` is the same as `urlValue`, at `1`
		// will be the service name we need, and at `2` the rest of the path.
		matcher := functionMatcher.Copy()
		matches := matcher.FindStringSubmatch(urlValue)
		chainMatcher := chainMatcher.Copy()
		chainMatches := chainMatcher.FindStringSubmatch(urlValue)
		log.Print(chainMatches)
		if len(matches) == hasPathCount{
			serviceName = matches[nameIndex]
		}else if len(chainMatches) == hasPathCount{
			serviceName = chainMatches[nameIndex]
		}
	}
	return strings.Trim(serviceName, "/")
}

// LoggingNotifier notifies a log about a request
type LoggingNotifier struct {
}

// Notify a log about a request
func (LoggingNotifier) Notify(method string, URL string, originalURL string, statusCode int, duration time.Duration) {
/*	if !strings.Contains(originalURL, "/h") {  // /healthz will not be recorded
		log.Printf("Forwarded [%s] to %s - [%d] - %fs", method, originalURL, statusCode, duration.Seconds())
	}*/
}


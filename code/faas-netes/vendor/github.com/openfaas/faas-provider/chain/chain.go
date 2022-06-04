// Package proxy provides a default function invocation proxy method for OpenFaaS providers.
//
// The function proxy logic is used by the Gateway when `direct_functions` is set to false.
// This means that the provider will direct call the function and return the results.  This
// involves resolving the function by name and then copying the result into the original HTTP
// request.
//
// openfaas-provider has implemented a standard HTTP HandlerFunc that will handle setting
// timeout values, parsing the request path, and copying the request/response correctly.
// 		bootstrapHandlers := bootTypes.FaaSHandlers{
// 			FunctionProxy:  proxy.NewHandlerFunc(timeout, resolver),
// 			DeleteHandler:  handlers.MakeDeleteHandler(clientset),
// 			DeployHandler:  handlers.MakeDeployHandler(clientset),
// 			FunctionReader: handlers.MakeFunctionReader(clientset),
// 			ReplicaReader:  handlers.MakeReplicaReader(clientset),
// 			ReplicaUpdater: handlers.MakeReplicaUpdater(clientset),
// 			InfoHandler:    handlers.MakeInfoHandler(),
// 		}
//
// proxy.NewHandlerFunc is optional, but does simplify the logic of your provider.
package chain

import (
	"io"
	"log"
	"io/ioutil"
	// "fmt"
	"strings"
	"encoding/json"
	"net"
	"net/http"
	"bytes"
	"net/url"
	"time"
	// "strings"
	"github.com/gorilla/mux"
	"github.com/openfaas/faas-provider/httputil"
	"github.com/openfaas/faas-provider/types"
)

const (
	watchdogPort       = "8080"
	defaultContentType = "text/plain"
	revisePort		   = "30303"
)


// BaseURLResolver URL resolver for proxy requests
// The FaaS provider implementation is responsible for providing the resolver function implementation.
// BaseURLResolver.Resolve will receive the function name and should return the URL of the
// function service.

type BaseURLResolver interface {
	Resolve(functionName string, CPU int) (url.URL, types.FunctionDockerMetadata, error)
	ColdStart(functionName string) (url.URL, types.FunctionDockerMetadata, error)

	ReturnPod(functionName string, serviceIP string,  CPU int)
	GetResource(job *Job, stage int) int
	GetChain(chainName string) ([]string)

}

type reviseReq struct{
	CPURatio int "json:CPURatio"
	DockerName string "json:DockerName"
	PodUID string "json:PodUID"
}

// NewHandlerFunc creates a standard http.HandlerFunc to proxy function requests.
// The returned http.HandlerFunc will ensure:
//
// 	- proper proxy request timeouts
// 	- proxy requests for GET, POST, PATCH, PUT, and DELETE
// 	- path parsing including support for extracing the function name, sub-paths, and query paremeters
// 	- passing and setting the `X-Forwarded-Host` and `X-Forwarded-For` headers
// 	- logging errors and proxy request timing to stdout
//
// Note that this will panic if `resolver` is nil.
func NewHandlerChain(config types.FaaSConfig, resolver BaseURLResolver) http.HandlerFunc {
	if resolver == nil {
		panic("NewHandlerFunc: empty proxy handler resolver, cannot be nil")
	}

	proxyClient := NewProxyClientFromConfig(config)
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			defer r.Body.Close()
		}

		switch r.Method {
		case http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodGet,
			http.MethodOptions,
			http.MethodHead:
			proxyRequest(w, r, proxyClient, resolver)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

// NewProxyClientFromConfig creates a new http.Client designed for proxying requests and enforcing
// certain minimum configuration values.
func NewProxyClientFromConfig(config types.FaaSConfig) *http.Client {
	return NewProxyClient(100 * time.Second, config.GetMaxIdleConns(), config.GetMaxIdleConnsPerHost())
}

// NewProxyClient creates a new http.Client designed for proxying requests, this is exposed as a
// convenience method for internal or advanced uses. Most people should use NewProxyClientFromConfig.
func NewProxyClient(timeout time.Duration, maxIdleConns int, maxIdleConnsPerHost int) *http.Client {
	return &http.Client{
		// these Transport values ensure that the http Client will eventually timeout and prevents
		// infinite retries. The default http.Client configure these timeouts.  The specific
		// values tuned via performance testing/benchmarking
		//
		// Additional context can be found at
		// - https://medium.com/@nate510/don-t-use-go-s-default-http-client-4804cb19f779
		// - https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/
		//
		// Additionally, these overrides for the default client enable re-use of connections and prevent
		// CoreDNS from rate limiting under high traffic
		//
		// See also two similar projects where this value was updated:
		// https://github.com/prometheus/prometheus/pull/3592
		// https://github.com/minio/minio/pull/5860
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   timeout,
				KeepAlive: 1 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          maxIdleConns,
			MaxIdleConnsPerHost:   maxIdleConnsPerHost,
			IdleConnTimeout:       120 * time.Millisecond,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1500 * time.Millisecond,
		},
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func Transformation(response *http.Response)  *http.Request{
	c, _ := http.NewRequest("POST", "", nil)
	body, _ := ioutil.ReadAll(response.Body)
	// log.Println(string(body))
	c.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	return c
}

// proxyRequest handles the actual resolution of and then request to the function service.
func proxyRequest(w http.ResponseWriter, originalReq *http.Request, proxyClient *http.Client, resolver BaseURLResolver) {
	// ctx := originalReq.Context()

	pathVars := mux.Vars(originalReq)

	chainName := pathVars["name"]
	if chainName == "" {
		httputil.Errorf(w, http.StatusBadRequest, "Provide function name in the request path")
		return
	}

	job := Job{}
	job.InitialJob(chainName, proxyClient, 700)
	// appendJob(&job)
	addNewJob()
	go watch(chainName)

	if len(job.DAG.AdjMap) == 0 {
		log.Printf("resolver error: no functionchains for %s: %s\n", chainName)
		httputil.Errorf(w, http.StatusServiceUnavailable, "No endpoints available for chain: %s.", chainName)
		return
	}

	var response *http.Response

	st := time.Now()

	response = job.Start(originalReq, resolver)

	log.Print(time.Since(st))

	if response.Body != nil {
		defer response.Body.Close()
	}

	clientHeader := w.Header()
	copyHeaders(clientHeader, &response.Header)
	w.Header().Set("Content-Type", getContentType(originalReq.Header, response.Header))

	w.WriteHeader(response.StatusCode)
	removeEnsureJob(&job)
	if response.Body != nil {
		io.Copy(w, response.Body)
		// log.Printf("io结果为: %s\n", w)
	}
}


func ReviseResource(docker types.FunctionDockerMetadata, resource int, proxyClient *http.Client){
	// start := time.Now()
	host := docker.NodeIP + ":" + revisePort + "/revise"
	podUID := strings.Replace(docker.UID, "-", "_", -1)
	dockerName := "k8s_" + docker.FunctionName + "_" + docker.PodName + "_" + docker.Namespace + "_" + docker.UID + "_0"

	requestBody := new(bytes.Buffer)
	revisereq := reviseReq{
		PodUID: podUID,
		DockerName: dockerName,
		CPURatio: resource,
	}

	if err := json.NewEncoder(requestBody).Encode(revisereq); err != nil {
        return
    }

	r, err := http.NewRequest("POST", host, requestBody)
	r.Header.Set("Content-Type", "application/json")
	
	_, err = proxyClient.Do(r)

	if err != nil {
		log.Println("出错了！")
	}

} 


// buildProxyRequest creates a request object for the proxy request, it will ensure that
// the original request headers are preserved as well as setting openfaas system headers
func buildProxyRequest(originalReq *http.Request, baseURL url.URL, extraPath string) (*http.Request, error) {

	host := baseURL.Host
	if baseURL.Port() == "" {
		host = baseURL.Host + ":" + watchdogPort
	}

	url := url.URL{
		Scheme:   baseURL.Scheme,
		Host:     host,
		Path:     extraPath,
		RawQuery: originalReq.URL.RawQuery,
	}

	upstreamReq, err := http.NewRequest(originalReq.Method, url.String(), nil)

	if err != nil {
		return nil, err
	}
	copyHeaders(upstreamReq.Header, &originalReq.Header)

	if len(originalReq.Host) > 0 && upstreamReq.Header.Get("X-Forwarded-Host") == "" {
		upstreamReq.Header["X-Forwarded-Host"] = []string{originalReq.Host}
	}
	if upstreamReq.Header.Get("X-Forwarded-For") == "" {
		upstreamReq.Header["X-Forwarded-For"] = []string{originalReq.RemoteAddr}
	}

	if originalReq.Body != nil {
		upstreamReq.Body = originalReq.Body

	}

	return upstreamReq, nil
}

// copyHeaders clones the header values from the source into the destination.
func copyHeaders(destination http.Header, source *http.Header) {
	for k, v := range *source {
		vClone := make([]string, len(v))
		copy(vClone, v)
		destination[k] = vClone
	}
}

// getContentType resolves the correct Content-Type for a proxied function.
func getContentType(request http.Header, proxyResponse http.Header) (headerContentType string) {
	responseHeader := proxyResponse.Get("Content-Type")
	requestHeader := request.Get("Content-Type")

	if len(responseHeader) > 0 {
		headerContentType = responseHeader
	} else if len(requestHeader) > 0 {
		headerContentType = requestHeader
	} else {
		headerContentType = defaultContentType
	}

	return headerContentType
}

// Copyright (c) Alex Ellis 2017. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package bootstrap

import (
	"fmt"
	"log"
	"net/http"
	"github.com/gorilla/mux"
	"github.com/openfaas/faas-provider/auth"
	"github.com/openfaas/faas-provider/types"
	"github.com/openfaas/faas-provider/chain"
	v1app "k8s.io/client-go/listers/apps/v1"
	v1core "k8s.io/client-go/listers/core/v1"

)

// NameExpression for a function / service
const NameExpression = "-a-zA-Z_0-9."

var r *mux.Router


// Mark this as a Golang "package"
func init() {
	r = mux.NewRouter()
}

// Router gives access to the underlying router for when new routes need to be added.
func Router() *mux.Router {
	return r
}

// Serve load your handlers into the correct OpenFaaS route spec. This function is blocking.
func Serve(handlers *types.FaaSHandlers, config *types.FaaSConfig, dlister v1app.DeploymentLister, nlister v1core.NodeLister) {
	chain.InitialDataStore()
	chain.InitialEnv(dlister, nlister)
	if config.EnableBasicAuth {
		reader := auth.ReadBasicAuthFromDisk{
			SecretMountPath: config.SecretMountPath,
		}

		credentials, err := reader.Read()
		if err != nil {
			log.Fatal(err)
		}

		handlers.FunctionReader = auth.DecorateWithBasicAuth(handlers.FunctionReader, credentials)
		handlers.DeployHandler = auth.DecorateWithBasicAuth(handlers.DeployHandler, credentials)
		handlers.DeleteHandler = auth.DecorateWithBasicAuth(handlers.DeleteHandler, credentials)
		handlers.UpdateHandler = auth.DecorateWithBasicAuth(handlers.UpdateHandler, credentials)
		handlers.ReplicaReader = auth.DecorateWithBasicAuth(handlers.ReplicaReader, credentials)
		handlers.ReplicaUpdater = auth.DecorateWithBasicAuth(handlers.ReplicaUpdater, credentials)
		handlers.InfoHandler = auth.DecorateWithBasicAuth(handlers.InfoHandler, credentials)
		handlers.SecretHandler = auth.DecorateWithBasicAuth(handlers.SecretHandler, credentials)
		handlers.LogHandler = auth.DecorateWithBasicAuth(handlers.LogHandler, credentials)
	}

	// System (auth) endpoints
	r.HandleFunc("/system/functions", handlers.FunctionReader).Methods("GET")
	r.HandleFunc("/system/functions", handlers.DeployHandler).Methods("POST")
	r.HandleFunc("/system/functions", handlers.DeleteHandler).Methods("DELETE")
	r.HandleFunc("/system/functions", handlers.UpdateHandler).Methods("PUT")

	r.HandleFunc("/system/function/{name:["+NameExpression+"]+}", handlers.ReplicaReader).Methods("GET")
	r.HandleFunc("/system/scale-function/{name:["+NameExpression+"]+}", handlers.ReplicaUpdater).Methods("POST")
	r.HandleFunc("/system/info", handlers.InfoHandler).Methods("GET")

	r.HandleFunc("/system/secrets", handlers.SecretHandler).Methods(http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete)
	r.HandleFunc("/system/logs", handlers.LogHandler).Methods(http.MethodGet)

	r.HandleFunc("/system/namespaces", handlers.ListNamespaceHandler).Methods("GET")

	// Open endpoints
	// main.go??????proxy.NewHandlerFunc(config.FaaSConfig, functionLookup)??????proxy
	r.HandleFunc("/function/{name:["+NameExpression+"]+}", handlers.FunctionProxy)
	r.HandleFunc("/function/{name:["+NameExpression+"]+}/", handlers.FunctionProxy)
	r.HandleFunc("/function/{name:["+NameExpression+"]+}/{params:.*}", handlers.FunctionProxy)

	// ??????????????????
	r.HandleFunc("/chain/{name:["+NameExpression+"]+}", handlers.ChainProxy)
	r.HandleFunc("/chain/{name:["+NameExpression+"]+}/", handlers.ChainProxy)
	r.HandleFunc("/chain/{name:["+NameExpression+"]+}/{params:.*}", handlers.ChainProxy)

	if handlers.HealthHandler != nil {
		r.HandleFunc("/healthz", handlers.HealthHandler).Methods("GET")
	}

	readTimeout := config.ReadTimeout
	writeTimeout := config.WriteTimeout

	tcpPort := 8080
	if config.TCPPort != nil {
		tcpPort = *config.TCPPort
	}

	s := &http.Server{
		Addr:           fmt.Sprintf(":%d", tcpPort),
		ReadTimeout:    readTimeout,
		WriteTimeout:   writeTimeout,
		MaxHeaderBytes: http.DefaultMaxHeaderBytes, // 1MB - can be overridden by setting Server.MaxHeaderBytes.
		Handler:        r,
	}
	chain.InitialTables()

	
	// go chain.CommunicateWithTrainer()
	// go chain.RegulAdjust() 

	// go chain.Reset()
	log.Fatal(s.ListenAndServe())
}

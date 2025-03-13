// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"k8s.io/klog/v2"
)

func main() {
	klog.Info("Starting webhook server...")

	r := mux.NewRouter()
	r.HandleFunc("/mutate", mutateHandler)

	// Start the HTTP server
	port := os.Getenv("WEBHOOK_PORT")
	if port == "" {
		port = "8443" // Default webhook server port
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	err := server.ListenAndServeTLS("/certs/tls.crt", "/certs/tls.key")
	if err != nil {
		klog.Fatalf("Failed to start webhook: %v", err)
	}
}

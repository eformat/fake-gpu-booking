package main

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"

	"github.com/eformat/gpu-config-plugin/pkg/api"
	"github.com/eformat/gpu-config-plugin/pkg/kube"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	port := envOr("PORT", "9443")
	distDir := envOr("PLUGIN_DIST_DIR", "./dist")
	gpuNS := envOr("GPU_OPERATOR_NAMESPACE", "gpu-operator")
	bookingNS := envOr("BOOKING_PLUGIN_NAMESPACE", "gpu-booking-app-plugin")
	certFile := envOr("TLS_CERT_FILE", "/var/serving-cert/tls.crt")
	keyFile := envOr("TLS_KEY_FILE", "/var/serving-cert/tls.key")

	if strings.EqualFold(os.Getenv("DEV_MODE"), "true") {
		api.DevMode = true
		slog.Warn("DEV_MODE enabled — anonymous admin access granted")
	}

	client, err := kube.NewClient(gpuNS, bookingNS)
	if err != nil {
		slog.Warn("k8s client not available", "error", err)
	}
	api.KubeClient = client

	r := mux.NewRouter()

	apiRouter := r.PathPrefix("/api").Subrouter()
	apiRouter.Use(api.RateLimitMiddleware)
	apiRouter.Use(api.AuthMiddleware)

	apiRouter.HandleFunc("/auth/me", api.MeHandler).Methods("GET")
	apiRouter.HandleFunc("/profiles", api.ProfilesHandler).Methods("GET")
	apiRouter.HandleFunc("/config", api.ConfigHandler).Methods("GET")
	apiRouter.HandleFunc("/status", api.StatusHandler).Methods("GET")
	apiRouter.HandleFunc("/deploy", api.DeployHandler).Methods("POST")
	apiRouter.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		api.JsonResponse(w, map[string]string{"status": "ok"})
	}).Methods("GET")

	r.PathPrefix("/").Handler(http.FileServer(http.Dir(distDir)))

	addr := fmt.Sprintf(":%s", port)
	slog.Info("starting server", "port", port, "distDir", distDir, "gpuOperatorNS", gpuNS)

	if fileExists(certFile) && fileExists(keyFile) {
		slog.Info("TLS enabled", "cert", certFile, "key", keyFile)
		tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
		server := &http.Server{Addr: addr, Handler: r, TLSConfig: tlsCfg}
		if err := server.ListenAndServeTLS(certFile, keyFile); err != nil {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	} else {
		slog.Warn("TLS disabled — cert/key not found")
		if err := http.ListenAndServe(addr, r); err != nil {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

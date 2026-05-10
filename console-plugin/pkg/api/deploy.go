package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/eformat/gpu-config-plugin/pkg/kube"
)

var KubeClient *kube.Client

func DeployHandler(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r)
	if !user.IsAdmin {
		HttpError(w, http.StatusForbidden, "admin access required")
		return
	}

	var req kube.DeployRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		HttpError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.GPUProduct == "" || req.GPUCount <= 0 {
		HttpError(w, http.StatusBadRequest, "gpuProduct and gpuCount are required")
		return
	}

	slog.Info("deploying GPU config", "user", user.Username, "profile", req.Profile, "product", req.GPUProduct)

	result := KubeClient.Deploy(r.Context(), &req)
	JsonResponse(w, result)
}

func ConfigHandler(w http.ResponseWriter, r *http.Request) {
	config, err := KubeClient.GetCurrentConfig(r.Context())
	if err != nil {
		HttpError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if config == nil {
		JsonResponse(w, map[string]any{"config": nil})
		return
	}
	JsonResponse(w, map[string]any{"config": config})
}

func StatusHandler(w http.ResponseWriter, r *http.Request) {
	status, err := KubeClient.GetStatus(r.Context())
	if err != nil {
		HttpError(w, http.StatusInternalServerError, err.Error())
		return
	}
	JsonResponse(w, status)
}

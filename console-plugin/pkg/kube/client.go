package kube

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"gopkg.in/yaml.v3"
)

type Client struct {
	clientset *kubernetes.Clientset
	namespace string
}

func NewClient(namespace string) (*Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("in-cluster config: %w", err)
	}
	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("clientset: %w", err)
	}
	return &Client{clientset: cs, namespace: namespace}, nil
}

type DeployRequest struct {
	Profile     string     `json:"profile"`
	GPUProduct  string     `json:"gpuProduct"`
	GPUCount    int        `json:"gpuCount"`
	GPUMemory   int        `json:"gpuMemory"`
	MIGStrategy string     `json:"migStrategy"`
	MIGSlices   []MIGSlice `json:"migSlices"`
}

type MIGSlice struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type StepResult struct {
	Step    string `json:"step"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type DeployResult struct {
	Steps []StepResult `json:"steps"`
}

func (c *Client) GetCurrentConfig(ctx context.Context) (*DeployRequest, error) {
	cm, err := c.clientset.CoreV1().ConfigMaps(c.namespace).Get(ctx, "gpu-config-selected", metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	data, ok := cm.Data["config.json"]
	if !ok {
		return nil, nil
	}
	var req DeployRequest
	if err := yaml.Unmarshal([]byte(data), &req); err != nil {
		return nil, err
	}
	return &req, nil
}

func (c *Client) Deploy(ctx context.Context, req *DeployRequest) *DeployResult {
	result := &DeployResult{}

	result.Steps = append(result.Steps, c.persistSelection(ctx, req))
	result.Steps = append(result.Steps, c.updateTopology(ctx, req))
	result.Steps = append(result.Steps, c.labelNodes(ctx, req))
	result.Steps = append(result.Steps, c.restartWorkloads(ctx))

	return result
}

func (c *Client) persistSelection(ctx context.Context, req *DeployRequest) StepResult {
	data, _ := yaml.Marshal(req)
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gpu-config-selected",
			Namespace: c.namespace,
		},
		Data: map[string]string{
			"config.json": string(data),
		},
	}
	existing, err := c.clientset.CoreV1().ConfigMaps(c.namespace).Get(ctx, "gpu-config-selected", metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			_, err = c.clientset.CoreV1().ConfigMaps(c.namespace).Create(ctx, cm, metav1.CreateOptions{})
		}
	} else {
		existing.Data = cm.Data
		_, err = c.clientset.CoreV1().ConfigMaps(c.namespace).Update(ctx, existing, metav1.UpdateOptions{})
	}
	if err != nil {
		slog.Error("persist selection failed", "error", err)
		return StepResult{Step: "persist", Status: "error", Message: err.Error()}
	}
	return StepResult{Step: "persist", Status: "ok"}
}

func (c *Client) updateTopology(ctx context.Context, req *DeployRequest) StepResult {
	topology := buildTopologyYAML(req)

	cm, err := c.clientset.CoreV1().ConfigMaps(c.namespace).Get(ctx, "topology", metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			newCM := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: "topology", Namespace: c.namespace},
				Data:       map[string]string{"topology.yml": topology},
			}
			_, err = c.clientset.CoreV1().ConfigMaps(c.namespace).Create(ctx, newCM, metav1.CreateOptions{})
		}
	} else {
		cm.Data["topology.yml"] = topology
		_, err = c.clientset.CoreV1().ConfigMaps(c.namespace).Update(ctx, cm, metav1.UpdateOptions{})
	}
	if err != nil {
		slog.Error("update topology failed", "error", err)
		return StepResult{Step: "topology", Status: "error", Message: err.Error()}
	}
	return StepResult{Step: "topology", Status: "ok"}
}

type topologyConfig struct {
	NodePools        map[string]nodePool `yaml:"nodePools"`
	NodePoolLabelKey string              `yaml:"nodePoolLabelKey"`
	MIGStrategy      string              `yaml:"migStrategy"`
}

type nodePool struct {
	GPUProduct   string        `yaml:"gpuProduct"`
	GPUCount     int           `yaml:"gpuCount"`
	GPUMemory    int           `yaml:"gpuMemory"`
	OtherDevices []otherDevice `yaml:"otherDevices,omitempty"`
}

type otherDevice struct {
	Name  string `yaml:"name"`
	Count int    `yaml:"count"`
}

func buildTopologyYAML(req *DeployRequest) string {
	pool := nodePool{
		GPUProduct: req.GPUProduct,
		GPUCount:   req.GPUCount,
		GPUMemory:  req.GPUMemory,
	}
	for _, s := range req.MIGSlices {
		if s.Count > 0 {
			pool.OtherDevices = append(pool.OtherDevices, otherDevice{Name: s.Name, Count: s.Count})
		}
	}
	tc := topologyConfig{
		NodePools:        map[string]nodePool{"default": pool},
		NodePoolLabelKey: "run.ai/simulated-gpu-node-pool",
		MIGStrategy:      req.MIGStrategy,
	}
	out, _ := yaml.Marshal(tc)
	return string(out)
}

func (c *Client) labelNodes(ctx context.Context, req *DeployRequest) StepResult {
	nodes, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{
		LabelSelector: "node-role.kubernetes.io/worker",
	})
	if err != nil {
		slog.Error("list nodes failed", "error", err)
		return StepResult{Step: "label-nodes", Status: "error", Message: err.Error()}
	}

	migEnabled := req.MIGStrategy == "mixed" && len(req.MIGSlices) > 0
	var migAnnotation string
	if migEnabled {
		migAnnotation = buildMIGAnnotation(req)
	}

	var errors []string
	for _, node := range nodes.Items {
		if err := c.labelNode(ctx, &node, migEnabled, migAnnotation); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", node.Name, err))
		}
	}
	if len(errors) > 0 {
		return StepResult{Step: "label-nodes", Status: "error", Message: strings.Join(errors, "; ")}
	}
	return StepResult{Step: "label-nodes", Status: "ok", Message: fmt.Sprintf("%d nodes labeled", len(nodes.Items))}
}

func (c *Client) labelNode(ctx context.Context, node *corev1.Node, migEnabled bool, migAnnotation string) error {
	var patchJSON string
	if migEnabled {
		escaped := strings.ReplaceAll(migAnnotation, `"`, `\"`)
		escaped = strings.ReplaceAll(escaped, "\n", `\n`)
		patchJSON = fmt.Sprintf(`{"metadata":{"labels":{"run.ai/simulated-gpu-node-pool":"default","node-role.kubernetes.io/runai-dynamic-mig":"true"},"annotations":{"run.ai/mig.config":"%s"}}}`, escaped)
	} else {
		patchJSON = `{"metadata":{"labels":{"run.ai/simulated-gpu-node-pool":"default","node-role.kubernetes.io/runai-dynamic-mig":null},"annotations":{"run.ai/mig.config":null}}}`
	}
	_, err := c.clientset.CoreV1().Nodes().Patch(ctx, node.Name, types.StrategicMergePatchType, []byte(patchJSON), metav1.PatchOptions{})
	return err
}

type migConfig struct {
	Version    string           `yaml:"version"`
	MIGConfigs migConfigConfigs `yaml:"mig-configs"`
}

type migConfigConfigs struct {
	Selected []migDeviceConfig `yaml:"selected"`
}

type migDeviceConfig struct {
	Devices    []string    `yaml:"devices"`
	MIGEnabled bool        `yaml:"mig-enabled"`
	MIGDevices []migDevice `yaml:"mig-devices"`
}

type migDevice struct {
	Name     string `yaml:"name"`
	Position int    `yaml:"position"`
	Size     int    `yaml:"size"`
}

func buildMIGAnnotation(req *DeployRequest) string {
	devices := buildMIGDeviceLayout(req.MIGSlices)

	var selected []migDeviceConfig
	for i := 0; i < req.GPUCount; i++ {
		selected = append(selected, migDeviceConfig{
			Devices:    []string{fmt.Sprintf("%d", i)},
			MIGEnabled: true,
			MIGDevices: devices,
		})
	}

	cfg := migConfig{
		Version:    "v1",
		MIGConfigs: migConfigConfigs{Selected: selected},
	}
	out, _ := yaml.Marshal(cfg)
	return string(out)
}

func buildMIGDeviceLayout(slices []MIGSlice) []migDevice {
	var devices []migDevice
	pos := 0
	for _, s := range slices {
		if s.Count <= 0 {
			continue
		}
		sliceName := strings.TrimPrefix(s.Name, "nvidia.com/mig-")
		size := migSliceSize(sliceName)
		perDevice := s.Count / 8
		if perDevice < 1 {
			perDevice = 1
		}
		for i := 0; i < perDevice; i++ {
			devices = append(devices, migDevice{
				Name:     sliceName,
				Position: pos,
				Size:     size,
			})
			pos += size
		}
	}
	return devices
}

func migSliceSize(name string) int {
	if strings.HasPrefix(name, "7g.") {
		return 7
	}
	if strings.HasPrefix(name, "4g.") {
		return 4
	}
	if strings.HasPrefix(name, "3g.") {
		return 3
	}
	if strings.HasPrefix(name, "2g.") {
		return 2
	}
	return 1
}

func (c *Client) restartWorkloads(ctx context.Context) StepResult {
	// Delete per-node topology ConfigMaps so status-updater regenerates them from the updated topology
	cmList, err := c.clientset.CoreV1().ConfigMaps(c.namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, cm := range cmList.Items {
			if strings.HasPrefix(cm.Name, "topology-") {
				c.clientset.CoreV1().ConfigMaps(c.namespace).Delete(ctx, cm.Name, metav1.DeleteOptions{})
			}
		}
	}

	restartAnnotation := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`, time.Now().Format(time.RFC3339))

	deployments := []string{"status-updater", "topology-server"}
	daemonsets := []string{"device-plugin", "nvidia-dcgm-exporter"}

	var errors []string
	for _, name := range deployments {
		_, err := c.clientset.AppsV1().Deployments(c.namespace).Patch(ctx, name, types.StrategicMergePatchType, []byte(restartAnnotation), metav1.PatchOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			errors = append(errors, fmt.Sprintf("deployment/%s: %v", name, err))
		}
	}
	for _, name := range daemonsets {
		_, err := c.clientset.AppsV1().DaemonSets(c.namespace).Patch(ctx, name, types.StrategicMergePatchType, []byte(restartAnnotation), metav1.PatchOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			errors = append(errors, fmt.Sprintf("daemonset/%s: %v", name, err))
		}
	}
	if len(errors) > 0 {
		return StepResult{Step: "restart", Status: "error", Message: strings.Join(errors, "; ")}
	}
	return StepResult{Step: "restart", Status: "ok"}
}

type ClusterStatus struct {
	Nodes       []NodeStatus `json:"nodes"`
	Deployments []WorkloadStatus `json:"deployments"`
	DaemonSets  []WorkloadStatus `json:"daemonSets"`
}

type NodeStatus struct {
	Name       string `json:"name"`
	Ready      bool   `json:"ready"`
	GPUPool    string `json:"gpuPool"`
	MIGEnabled bool   `json:"migEnabled"`
}

type WorkloadStatus struct {
	Name     string `json:"name"`
	Ready    int32  `json:"ready"`
	Desired  int32  `json:"desired"`
}

func (c *Client) GetStatus(ctx context.Context) (*ClusterStatus, error) {
	status := &ClusterStatus{}

	nodes, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{
		LabelSelector: "node-role.kubernetes.io/worker",
	})
	if err != nil {
		return nil, err
	}
	for _, n := range nodes.Items {
		ready := false
		for _, cond := range n.Status.Conditions {
			if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
				ready = true
			}
		}
		_, migEnabled := n.Labels["node-role.kubernetes.io/runai-dynamic-mig"]
		status.Nodes = append(status.Nodes, NodeStatus{
			Name:       n.Name,
			Ready:      ready,
			GPUPool:    n.Labels["run.ai/simulated-gpu-node-pool"],
			MIGEnabled: migEnabled,
		})
	}

	deps, err := c.clientset.AppsV1().Deployments(c.namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, d := range deps.Items {
			status.Deployments = append(status.Deployments, WorkloadStatus{
				Name:    d.Name,
				Ready:   d.Status.ReadyReplicas,
				Desired: *d.Spec.Replicas,
			})
		}
	}

	dss, err := c.clientset.AppsV1().DaemonSets(c.namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, ds := range dss.Items {
			status.DaemonSets = append(status.DaemonSets, WorkloadStatus{
				Name:    ds.Name,
				Ready:   ds.Status.NumberReady,
				Desired: ds.Status.DesiredNumberScheduled,
			})
		}
	}

	return status, nil
}

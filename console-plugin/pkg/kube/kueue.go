package kube

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

var (
	cohortGVR = schema.GroupVersionResource{
		Group: "kueue.x-k8s.io", Version: "v1beta1", Resource: "cohorts",
	}
	clusterQueueGVR = schema.GroupVersionResource{
		Group: "kueue.x-k8s.io", Version: "v1beta1", Resource: "clusterqueues",
	}
	resourceFlavorGVR = schema.GroupVersionResource{
		Group: "kueue.x-k8s.io", Version: "v1beta1", Resource: "resourceflavors",
	}
	hardwareProfileGVR = schema.GroupVersionResource{
		Group: "dashboard.opendatahub.io", Version: "v1", Resource: "hardwareprofiles",
	}
)

const managedLabel = "gpu-config-plugin.openshift.io/managed"

func (c *Client) updateKueueResources(ctx context.Context, req *DeployRequest) StepResult {
	totalCPU, totalMemGi, err := c.getClusterCapacity(ctx)
	if err != nil {
		return StepResult{Step: "kueue", Status: "error", Message: fmt.Sprintf("get cluster capacity: %v", err)}
	}

	numNodes, err := c.getWorkerNodeCount(ctx)
	if err != nil {
		return StepResult{Step: "kueue", Status: "error", Message: fmt.Sprintf("get node count: %v", err)}
	}

	gpuCount := req.GPUCount * numNodes
	migCounts := map[string]int{}
	for _, s := range req.MIGSlices {
		if s.Count > 0 {
			migCounts[s.Name] = s.Count * numNodes
		}
	}

	var errors []string

	if err := c.ensureResourceFlavor(ctx); err != nil {
		errors = append(errors, fmt.Sprintf("resourceflavor: %v", err))
	}

	if err := c.updateCohort(ctx, totalCPU, totalMemGi, gpuCount, migCounts); err != nil {
		errors = append(errors, fmt.Sprintf("cohort: %v", err))
	}

	if err := c.updateClusterQueue(ctx, "unreserved", gpuCount, migCounts); err != nil {
		errors = append(errors, fmt.Sprintf("cq/unreserved: %v", err))
	}

	if err := c.updateClusterQueue(ctx, "unreserved-priority", gpuCount, migCounts); err != nil {
		errors = append(errors, fmt.Sprintf("cq/unreserved-priority: %v", err))
	}

	if err := c.updateHardwareProfiles(ctx, req, gpuCount, migCounts); err != nil {
		errors = append(errors, fmt.Sprintf("hardwareprofiles: %v", err))
	}

	if len(errors) > 0 {
		return StepResult{Step: "kueue", Status: "error", Message: strings.Join(errors, "; ")}
	}
	return StepResult{Step: "kueue", Status: "ok"}
}

func (c *Client) getClusterCapacity(ctx context.Context) (cpuCores int, memGi int, err error) {
	nodes, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{
		LabelSelector: "node-role.kubernetes.io/worker",
	})
	if err != nil {
		return 0, 0, err
	}
	for _, n := range nodes.Items {
		cpuCores += int(n.Status.Allocatable.Cpu().Value())
		memGi += int(n.Status.Allocatable.Memory().Value() / (1024 * 1024 * 1024))
	}
	return
}

func (c *Client) getWorkerNodeCount(ctx context.Context) (int, error) {
	nodes, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{
		LabelSelector: "node-role.kubernetes.io/worker",
	})
	if err != nil {
		return 0, err
	}
	return len(nodes.Items), nil
}

func (c *Client) ensureResourceFlavor(ctx context.Context) error {
	flavor := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "kueue.x-k8s.io/v1beta1",
			"kind":       "ResourceFlavor",
			"metadata": map[string]interface{}{
				"name": "gpu-pool",
				"labels": map[string]interface{}{
					managedLabel: "true",
				},
			},
			"spec": map[string]interface{}{},
		},
	}
	_, err := c.dynamicClient.Resource(resourceFlavorGVR).Get(ctx, "gpu-pool", metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = c.dynamicClient.Resource(resourceFlavorGVR).Create(ctx, flavor, metav1.CreateOptions{})
	}
	return err
}

func (c *Client) updateCohort(ctx context.Context, totalCPU, totalMemGi, gpuCount int, migCounts map[string]int) error {
	resourceGroups := c.buildResourceGroups("gpu-pool", totalCPU, totalMemGi, gpuCount, migCounts)

	patch := map[string]interface{}{
		"apiVersion": "kueue.x-k8s.io/v1beta1",
		"kind":       "Cohort",
		"metadata": map[string]interface{}{
			"name": "unreserved",
		},
		"spec": map[string]interface{}{
			"resourceGroups": resourceGroups,
		},
	}

	data, _ := marshalJSON(patch)
	_, err := c.dynamicClient.Resource(cohortGVR).Patch(ctx, "unreserved", types.MergePatchType, data, metav1.PatchOptions{})
	if apierrors.IsNotFound(err) {
		obj := &unstructured.Unstructured{Object: patch}
		_, err = c.dynamicClient.Resource(cohortGVR).Create(ctx, obj, metav1.CreateOptions{})
	}
	return err
}

func (c *Client) updateClusterQueue(ctx context.Context, name string, gpuCount int, migCounts map[string]int) error {
	resourceGroups := c.buildResourceGroups("gpu-pool", 0, 0, 0, zeroedMigCounts(migCounts))

	patch := map[string]interface{}{
		"spec": map[string]interface{}{
			"resourceGroups": resourceGroups,
		},
	}

	data, _ := marshalJSON(patch)
	_, err := c.dynamicClient.Resource(clusterQueueGVR).Patch(ctx, name, types.MergePatchType, data, metav1.PatchOptions{})
	return err
}

func (c *Client) buildResourceGroups(flavorName string, totalCPU, totalMemGi, gpuCount int, migCounts map[string]int) []interface{} {
	coveredResources := []interface{}{"cpu", "memory", "nvidia.com/gpu"}
	resources := []interface{}{
		map[string]interface{}{"name": "cpu", "nominalQuota": fmt.Sprintf("%d", totalCPU)},
		map[string]interface{}{"name": "memory", "nominalQuota": fmt.Sprintf("%dGi", totalMemGi)},
		map[string]interface{}{"name": "nvidia.com/gpu", "nominalQuota": fmt.Sprintf("%d", gpuCount)},
	}

	migNames := sortedKeys(migCounts)
	for _, name := range migNames {
		count := migCounts[name]
		coveredResources = append(coveredResources, name)
		resources = append(resources, map[string]interface{}{
			"name": name, "nominalQuota": fmt.Sprintf("%d", count),
		})
	}

	return []interface{}{
		map[string]interface{}{
			"coveredResources": coveredResources,
			"flavors": []interface{}{
				map[string]interface{}{
					"name":      flavorName,
					"resources": resources,
				},
			},
		},
	}
}

func (c *Client) updateHardwareProfiles(ctx context.Context, req *DeployRequest, gpuCount int, migCounts map[string]int) error {
	existing, err := c.dynamicClient.Resource(hardwareProfileGVR).Namespace("redhat-ods-applications").List(ctx, metav1.ListOptions{
		LabelSelector: managedLabel + "=true",
	})
	if err == nil {
		for _, item := range existing.Items {
			c.dynamicClient.Resource(hardwareProfileGVR).Namespace("redhat-ods-applications").Delete(ctx, item.GetName(), metav1.DeleteOptions{})
		}
	}

	profiles := []struct {
		name        string
		displayName string
		identifier  string
		maxCount    int
	}{
		{
			name:        "unreserved-gpu",
			displayName: fmt.Sprintf("Unreserved %s GPU", req.GPUProduct),
			identifier:  "nvidia.com/gpu",
			maxCount:    gpuCount,
		},
	}
	for migName, count := range migCounts {
		normalized := strings.TrimPrefix(migName, "nvidia.com/mig-")
		profiles = append(profiles, struct {
			name        string
			displayName string
			identifier  string
			maxCount    int
		}{
			name:        fmt.Sprintf("unreserved-mig-%s", strings.ReplaceAll(normalized, ".", "-")),
			displayName: fmt.Sprintf("Unreserved MIG %s", normalized),
			identifier:  migName,
			maxCount:    count,
		})
	}

	for _, p := range profiles {
		hp := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "dashboard.opendatahub.io/v1",
				"kind":       "HardwareProfile",
				"metadata": map[string]interface{}{
					"name":      p.name,
					"namespace": "redhat-ods-applications",
					"labels": map[string]interface{}{
						managedLabel: "true",
					},
				},
				"spec": map[string]interface{}{
					"displayName": p.displayName,
					"description": fmt.Sprintf("GPU resource: %s (max %d)", p.identifier, p.maxCount),
					"enabled":     true,
					"identifiers": []interface{}{
						map[string]interface{}{
							"displayName": p.displayName,
							"identifier":  p.identifier,
							"maxCount":    int64(p.maxCount),
							"minCount":    int64(1),
						},
					},
					"nodeSelectors": []interface{}{},
					"tolerations":   []interface{}{},
				},
			},
		}
		_, err := c.dynamicClient.Resource(hardwareProfileGVR).Namespace("redhat-ods-applications").Create(ctx, hp, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			_, err = c.dynamicClient.Resource(hardwareProfileGVR).Namespace("redhat-ods-applications").Update(ctx, hp, metav1.UpdateOptions{})
		}
		if err != nil {
			slog.Error("hardware profile create/update failed", "name", p.name, "error", err)
		}
	}
	return nil
}

func (c *Client) restartBookingPlugin(ctx context.Context) StepResult {
	if c.bookingNS == "" {
		return StepResult{Step: "restart-booking", Status: "ok", Message: "skipped (no booking namespace)"}
	}
	restartAnnotation := fmt.Sprintf(
		`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`,
		time.Now().Format(time.RFC3339),
	)
	_, err := c.clientset.AppsV1().Deployments(c.bookingNS).Patch(
		ctx, "gpu-booking-plugin", types.StrategicMergePatchType,
		[]byte(restartAnnotation), metav1.PatchOptions{},
	)
	if err != nil && !apierrors.IsNotFound(err) {
		return StepResult{Step: "restart-booking", Status: "error", Message: err.Error()}
	}
	return StepResult{Step: "restart-booking", Status: "ok"}
}

func zeroedMigCounts(m map[string]int) map[string]int {
	z := make(map[string]int, len(m))
	for k := range m {
		z[k] = 0
	}
	return z
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func marshalJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

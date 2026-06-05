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
		Group: "kueue.x-k8s.io", Version: "v1beta2", Resource: "cohorts",
	}
	clusterQueueGVR = schema.GroupVersionResource{
		Group: "kueue.x-k8s.io", Version: "v1beta2", Resource: "clusterqueues",
	}
	resourceFlavorGVR = schema.GroupVersionResource{
		Group: "kueue.x-k8s.io", Version: "v1beta2", Resource: "resourceflavors",
	}
	hardwareProfileGVR = schema.GroupVersionResource{
		Group: "infrastructure.opendatahub.io", Version: "v1", Resource: "hardwareprofiles",
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

	if err := c.deleteStaleResourceFlavors(ctx); err != nil {
		errors = append(errors, fmt.Sprintf("cleanup-flavors: %v", err))
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
		LabelSelector: "run.ai/simulated-gpu-node-pool=default",
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
		LabelSelector: "run.ai/simulated-gpu-node-pool=default",
	})
	if err != nil {
		return 0, err
	}
	return len(nodes.Items), nil
}

func (c *Client) ensureResourceFlavor(ctx context.Context) error {
	flavor := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "kueue.x-k8s.io/v1beta2",
			"kind":       "ResourceFlavor",
			"metadata": map[string]interface{}{
				"name": "gpu-pool",
				"labels": map[string]interface{}{
					managedLabel: "true",
				},
			},
			"spec": map[string]interface{}{
				"nodeLabels": map[string]interface{}{
					"run.ai/simulated-gpu-node-pool": "default",
				},
			},
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

	existing, err := c.dynamicClient.Resource(cohortGVR).Get(ctx, "unreserved", metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		obj := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "kueue.x-k8s.io/v1beta2",
				"kind":       "Cohort",
				"metadata":   map[string]interface{}{"name": "unreserved"},
				"spec":       map[string]interface{}{"resourceGroups": resourceGroups},
			},
		}
		_, err = c.dynamicClient.Resource(cohortGVR).Create(ctx, obj, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	unstructured.SetNestedSlice(existing.Object, resourceGroups, "spec", "resourceGroups")
	_, err = c.dynamicClient.Resource(cohortGVR).Update(ctx, existing, metav1.UpdateOptions{})
	return err
}

func (c *Client) updateClusterQueue(ctx context.Context, name string, gpuCount int, migCounts map[string]int) error {
	resourceGroups := c.buildResourceGroups("gpu-pool", 0, 0, 0, zeroedMigCounts(migCounts))

	existing, err := c.dynamicClient.Resource(clusterQueueGVR).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		spec := map[string]interface{}{
			"cohortName":        "unreserved",
			"resourceGroups":    resourceGroups,
			"namespaceSelector": map[string]interface{}{},
		}
		if name == "unreserved-priority" {
			spec["namespaceSelector"] = map[string]interface{}{
				"matchLabels": map[string]interface{}{
					"rhai-tmm.dev/clusterqueues": "h200-priority",
				},
			}
			spec["preemption"] = map[string]interface{}{
				"borrowWithinCohort": map[string]interface{}{
					"maxPriorityThreshold": int64(100),
					"policy":               "LowerPriority",
				},
				"reclaimWithinCohort": "LowerPriority",
			}
		}
		obj := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "kueue.x-k8s.io/v1beta2",
				"kind":       "ClusterQueue",
				"metadata":   map[string]interface{}{"name": name},
				"spec":       spec,
			},
		}
		_, err = c.dynamicClient.Resource(clusterQueueGVR).Create(ctx, obj, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	unstructured.SetNestedSlice(existing.Object, resourceGroups, "spec", "resourceGroups")
	if _, found, _ := unstructured.NestedMap(existing.Object, "spec", "namespaceSelector"); !found {
		unstructured.SetNestedMap(existing.Object, map[string]interface{}{}, "spec", "namespaceSelector")
	}
	if _, found, _ := unstructured.NestedString(existing.Object, "spec", "cohortName"); !found {
		unstructured.SetNestedField(existing.Object, "unreserved", "spec", "cohortName")
	}
	_, err = c.dynamicClient.Resource(clusterQueueGVR).Update(ctx, existing, metav1.UpdateOptions{})
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
		// Strip "Xg." prefix to get just the memory part (e.g., "1g.18gb" → "18gb")
		shortName := normalized
		if idx := strings.Index(normalized, "."); idx >= 0 {
			shortName = normalized[idx+1:]
		}
		profiles = append(profiles, struct {
			name        string
			displayName string
			identifier  string
			maxCount    int
		}{
			name:        fmt.Sprintf("unreserved-mig-%s", shortName),
			displayName: fmt.Sprintf("Unreserved MIG %s", normalized),
			identifier:  migName,
			maxCount:    count,
		})
	}

	for _, p := range profiles {
		hp := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "infrastructure.opendatahub.io/v1",
				"kind":       "HardwareProfile",
				"metadata": map[string]interface{}{
					"name":      p.name,
					"namespace": "redhat-ods-applications",
					"labels": map[string]interface{}{
						managedLabel: "true",
					},
				},
				"spec": map[string]interface{}{
					"identifiers": []interface{}{
						map[string]interface{}{
							"displayName":  p.displayName,
							"identifier":   p.identifier,
							"defaultCount": int64(1),
							"maxCount":     int64(p.maxCount),
							"minCount":     int64(1),
						},
					},
					"scheduling": map[string]interface{}{
						"type": "Queue",
						"kueue": map[string]interface{}{
							"localQueueName": "unreserved",
							"priorityClass":  "None",
						},
					},
				},
			},
		}
		_, err := c.dynamicClient.Resource(hardwareProfileGVR).Namespace("redhat-ods-applications").Create(ctx, hp, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			existing, getErr := c.dynamicClient.Resource(hardwareProfileGVR).Namespace("redhat-ods-applications").Get(ctx, p.name, metav1.GetOptions{})
			if getErr == nil {
				hp.SetResourceVersion(existing.GetResourceVersion())
				_, err = c.dynamicClient.Resource(hardwareProfileGVR).Namespace("redhat-ods-applications").Update(ctx, hp, metav1.UpdateOptions{})
			} else {
				err = getErr
			}
		}
		if err != nil {
			slog.Error("hardware profile create/update failed", "name", p.name, "error", err)
		}
	}
	return nil
}

func (c *Client) rolloutRestartBookingPlugin(ctx context.Context) StepResult {
	if c.bookingNS == "" {
		return StepResult{Step: "restart-booking", Status: "ok", Message: "skipped"}
	}
	patch := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`, time.Now().Format(time.RFC3339))
	_, err := c.clientset.AppsV1().Deployments(c.bookingNS).Patch(
		ctx, "gpu-booking-plugin", types.StrategicMergePatchType,
		[]byte(patch), metav1.PatchOptions{},
	)
	if err != nil && !apierrors.IsNotFound(err) {
		return StepResult{Step: "restart-booking", Status: "error", Message: err.Error()}
	}
	return StepResult{Step: "restart-booking", Status: "ok"}
}

func (c *Client) deleteStaleResourceFlavors(ctx context.Context) error {
	flavors, err := c.dynamicClient.Resource(resourceFlavorGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, f := range flavors.Items {
		name := f.GetName()
		if name == "gpu-pool" || name == "default-flavor" {
			continue
		}
		labels := f.GetLabels()
		spec := f.Object["spec"]
		specMap, isMap := spec.(map[string]interface{})
		hasGPUConfig := spec != nil && (!isMap || len(specMap) > 0)
		isManaged := labels != nil && labels[managedLabel] == "true"
		if hasGPUConfig || isManaged {
			slog.Info("deleting stale ResourceFlavor", "name", name)
			c.dynamicClient.Resource(resourceFlavorGVR).Delete(ctx, name, metav1.DeleteOptions{})
		}
	}
	return nil
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

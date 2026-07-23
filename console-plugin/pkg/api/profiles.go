package api

import (
	"net/http"
)

type MIGSlice struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type GPUProfile struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Product      string     `json:"product"`
	Architecture string     `json:"architecture"`
	Memory       string     `json:"memory"`
	GPUCount     int        `json:"gpuCount"`
	GPUMemoryMB  int        `json:"gpuMemoryMB"`
	MIGSupport   bool       `json:"migSupport"`
	MIGFamily    string     `json:"migFamily"`
	MIGSlices    []MIGSlice `json:"migSlices"`
}

type MIGFamily struct {
	ID     string   `json:"id"`
	Match  string   `json:"match"`
	Slices []string `json:"slices"`
}

var MIGFamilies = []MIGFamily{
	{
		ID:    "40gb",
		Match: "*40GB* (A100)",
		Slices: []string{
			"1g.5gb", "1g.10gb", "2g.10gb",
			"3g.20gb", "4g.20gb", "7g.40gb",
		},
	},
	{
		ID:    "80gb",
		Match: "*80GB* (H100)",
		Slices: []string{
			"1g.10gb", "1g.20gb", "2g.20gb",
			"3g.40gb", "4g.40gb", "7g.80gb",
		},
	},
	{
		ID:    "h200",
		Match: "*H200*",
		Slices: []string{
			"1g.18gb", "2g.35gb", "3g.71gb",
		},
	},
	{
		ID:    "gb200",
		Match: "*GB200*",
		Slices: []string{
			"1g.23gb", "1g.47gb", "2g.47gb",
			"3g.93gb", "4g.93gb", "7g.189gb",
		},
	},
	{
		ID:    "gb300",
		Match: "*GB300*",
		Slices: []string{
			"1g.35gb", "1g.70gb", "2g.70gb",
			"3g.139gb", "4g.139gb", "7g.278gb",
		},
	},
	{
		ID:    "vera-rubin",
		Match: "*Vera Rubin*",
		Slices: []string{
			"1g.35gb", "1g.70gb", "2g.70gb",
			"3g.139gb", "4g.139gb", "7g.278gb",
		},
	},
}

var BuiltinProfiles = []GPUProfile{
	{
		ID: "a100", Name: "NVIDIA A100-SXM4-40GB", Product: "NVIDIA-A100-SXM4-40GB",
		Architecture: "Ampere", Memory: "40 GiB HBM2e",
		GPUCount: 8, GPUMemoryMB: 40960, MIGSupport: true, MIGFamily: "40gb",
		MIGSlices: []MIGSlice{
			{Name: "nvidia.com/mig-1g.5gb", Count: 56},
			{Name: "nvidia.com/mig-2g.10gb", Count: 16},
			{Name: "nvidia.com/mig-3g.20gb", Count: 16},
			{Name: "nvidia.com/mig-7g.40gb", Count: 8},
		},
	},
	{
		ID: "h100", Name: "NVIDIA H100 80GB HBM3", Product: "NVIDIA-H100-80GB-HBM3",
		Architecture: "Hopper", Memory: "80 GiB HBM3",
		GPUCount: 8, GPUMemoryMB: 81920, MIGSupport: true, MIGFamily: "80gb",
		MIGSlices: []MIGSlice{
			{Name: "nvidia.com/mig-1g.10gb", Count: 56},
			{Name: "nvidia.com/mig-2g.20gb", Count: 16},
			{Name: "nvidia.com/mig-3g.40gb", Count: 16},
			{Name: "nvidia.com/mig-7g.80gb", Count: 8},
		},
	},
	{
		ID: "h200", Name: "NVIDIA H200", Product: "NVIDIA-H200",
		Architecture: "Hopper", Memory: "141 GiB HBM3e",
		GPUCount: 8, GPUMemoryMB: 143771, MIGSupport: true, MIGFamily: "h200",
		MIGSlices: []MIGSlice{
			{Name: "nvidia.com/mig-1g.18gb", Count: 16},
			{Name: "nvidia.com/mig-2g.35gb", Count: 8},
			{Name: "nvidia.com/mig-3g.71gb", Count: 8},
		},
	},
	{
		ID: "b200", Name: "NVIDIA B200", Product: "NVIDIA-B200",
		Architecture: "Blackwell", Memory: "192 GiB HBM3e",
		GPUCount: 8, GPUMemoryMB: 196608, MIGSupport: true, MIGFamily: "80gb",
		MIGSlices: []MIGSlice{
			{Name: "nvidia.com/mig-1g.10gb", Count: 56},
			{Name: "nvidia.com/mig-2g.20gb", Count: 16},
			{Name: "nvidia.com/mig-3g.40gb", Count: 16},
			{Name: "nvidia.com/mig-7g.80gb", Count: 8},
		},
	},
	{
		ID: "gb200", Name: "NVIDIA GB200 NVL", Product: "NVIDIA-GB200-NVL",
		Architecture: "Blackwell", Memory: "192 GiB HBM3e",
		GPUCount: 8, GPUMemoryMB: 196608, MIGSupport: true, MIGFamily: "gb200",
		MIGSlices: []MIGSlice{
			{Name: "nvidia.com/mig-1g.23gb", Count: 56},
			{Name: "nvidia.com/mig-2g.47gb", Count: 16},
			{Name: "nvidia.com/mig-3g.93gb", Count: 16},
			{Name: "nvidia.com/mig-7g.189gb", Count: 8},
		},
	},
	{
		ID: "gb300", Name: "NVIDIA GB300 NVL", Product: "NVIDIA-GB300-NVL",
		Architecture: "Blackwell Ultra", Memory: "288 GiB HBM3e",
		GPUCount: 8, GPUMemoryMB: 294912, MIGSupport: true, MIGFamily: "gb300",
		MIGSlices: []MIGSlice{
			{Name: "nvidia.com/mig-1g.35gb", Count: 56},
			{Name: "nvidia.com/mig-2g.70gb", Count: 16},
			{Name: "nvidia.com/mig-3g.139gb", Count: 16},
			{Name: "nvidia.com/mig-7g.278gb", Count: 8},
		},
	},
	{
		ID: "vera-rubin", Name: "NVIDIA Vera Rubin NVL", Product: "NVIDIA-Vera-Rubin-NVL",
		Architecture: "Rubin", Memory: "288 GiB HBM4",
		GPUCount: 8, GPUMemoryMB: 294912, MIGSupport: true, MIGFamily: "vera-rubin",
		MIGSlices: []MIGSlice{
			{Name: "nvidia.com/mig-1g.35gb", Count: 56},
			{Name: "nvidia.com/mig-2g.70gb", Count: 16},
			{Name: "nvidia.com/mig-3g.139gb", Count: 16},
			{Name: "nvidia.com/mig-7g.278gb", Count: 8},
		},
	},
	{
		ID: "l40s", Name: "NVIDIA L40S", Product: "NVIDIA-L40S",
		Architecture: "Ada Lovelace", Memory: "48 GiB GDDR6",
		GPUCount: 8, GPUMemoryMB: 49152, MIGSupport: false, MIGFamily: "",
		MIGSlices: nil,
	},
	{
		ID: "t4", Name: "NVIDIA T4", Product: "NVIDIA-T4",
		Architecture: "Turing", Memory: "16 GiB GDDR6",
		GPUCount: 8, GPUMemoryMB: 16384, MIGSupport: false, MIGFamily: "",
		MIGSlices: nil,
	},
}

func ProfilesHandler(w http.ResponseWriter, r *http.Request) {
	JsonResponse(w, map[string]any{
		"profiles":    BuiltinProfiles,
		"migFamilies": MIGFamilies,
	})
}

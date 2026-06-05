export interface MIGSlice {
  name: string;
  count: number;
}

export interface GPUProfile {
  id: string;
  name: string;
  product: string;
  architecture: string;
  memory: string;
  gpuCount: number;
  gpuMemoryMB: number;
  migSupport: boolean;
  migFamily: string;
  migSlices: MIGSlice[] | null;
}

export interface MIGFamily {
  id: string;
  match: string;
  slices: string[];
}

export interface DeployRequest {
  profile: string;
  gpuProduct: string;
  gpuCount: number;
  gpuMemory: number;
  migStrategy: string;
  migSlices: MIGSlice[];
  targetNodes: string[];
}

export interface StepResult {
  step: string;
  status: string;
  message?: string;
}

export interface DeployResult {
  steps: StepResult[];
}

export interface NodeStatus {
  name: string;
  ready: boolean;
  gpuPool: string;
  migEnabled: boolean;
}

export interface WorkloadStatus {
  name: string;
  ready: number;
  desired: number;
}

export interface ClusterStatus {
  nodes: NodeStatus[];
  deployments: WorkloadStatus[];
  daemonSets: WorkloadStatus[];
}

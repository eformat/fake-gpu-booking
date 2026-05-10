import { GPUProfile, MIGFamily, DeployRequest, DeployResult, ClusterStatus } from './constants';

const PLUGIN_NAME = 'gpu-config-plugin';
const PROXY_BASE = `/api/proxy/plugin/${PLUGIN_NAME}/backend/api`;

export interface AuthInfo {
  username: string;
  groups: string[];
  is_admin: boolean;
}

function getCSRFToken(): string {
  const match = document.cookie.match(/(?:^|;\s*)csrf-token=([^;]*)/);
  return match ? match[1] : '';
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const url = `${PROXY_BASE}${path}`;
  const headers: Record<string, string> = {
    'X-CSRFToken': getCSRFToken(),
    ...(options.headers as Record<string, string> || {}),
  };
  if (options.body && typeof options.body === 'string') {
    headers['Content-Type'] = 'application/json';
  }
  const res = await fetch(url, { ...options, headers });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || res.statusText);
  }
  return res.json();
}

export function getAuthInfo(): Promise<AuthInfo> {
  return request<AuthInfo>('/auth/me');
}

export function getProfiles(): Promise<{ profiles: GPUProfile[]; migFamilies: MIGFamily[] }> {
  return request('/profiles');
}

export function getCurrentConfig(): Promise<{ config: DeployRequest | null }> {
  return request('/config');
}

export function getStatus(): Promise<ClusterStatus> {
  return request('/status');
}

export function deployProfile(req: DeployRequest): Promise<DeployResult> {
  return request('/deploy', {
    method: 'POST',
    body: JSON.stringify(req),
  });
}

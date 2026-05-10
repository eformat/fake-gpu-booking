import * as React from 'react';
import { GPUProfile, MIGFamily, DeployRequest, ClusterStatus } from './constants';
import { getProfiles, getCurrentConfig, getStatus } from './api';

export function useProfiles() {
  const [profiles, setProfiles] = React.useState<GPUProfile[]>([]);
  const [migFamilies, setMigFamilies] = React.useState<MIGFamily[]>([]);
  const [loading, setLoading] = React.useState(true);

  React.useEffect(() => {
    getProfiles().then((data) => {
      setProfiles(data.profiles);
      setMigFamilies(data.migFamilies);
      setLoading(false);
    }).catch(() => setLoading(false));
  }, []);

  return { profiles, migFamilies, loading };
}

export function useCurrentConfig() {
  const [config, setConfig] = React.useState<DeployRequest | null>(null);
  const [loading, setLoading] = React.useState(true);

  const fetchConfig = React.useCallback(() => {
    getCurrentConfig().then((data) => {
      setConfig(data.config);
      setLoading(false);
    }).catch(() => setLoading(false));
  }, []);

  React.useEffect(() => {
    fetchConfig();
    const interval = setInterval(fetchConfig, 15000);
    return () => clearInterval(interval);
  }, [fetchConfig]);

  return { config, loading, fetchConfig };
}

export function useClusterStatus() {
  const [status, setStatus] = React.useState<ClusterStatus | null>(null);

  const fetchStatus = React.useCallback(() => {
    getStatus().then(setStatus).catch(() => {});
  }, []);

  React.useEffect(() => {
    fetchStatus();
    const interval = setInterval(fetchStatus, 10000);
    return () => clearInterval(interval);
  }, [fetchStatus]);

  return { status, fetchStatus };
}

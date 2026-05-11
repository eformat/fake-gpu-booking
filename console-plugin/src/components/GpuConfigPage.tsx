import * as React from 'react';
import { Helmet } from 'react-helmet';
import {
  PageSection,
  Title, Button,
  Tabs, Tab, TabTitleText,
  Gallery,
  Toolbar, ToolbarContent, ToolbarItem,
} from '@patternfly/react-core';
import { SyncIcon } from '@patternfly/react-icons';
import { useAuth } from '../utils/AuthContext';
import { useProfiles, useCurrentConfig, useClusterStatus } from '../utils/hooks';
import { DeployRequest, MIGSlice } from '../utils/constants';
import ProfileCard from './ProfileCard';
import MigConfigPanel from './MigConfigPanel';
import CustomProfileForm, { CustomConfig } from './CustomProfileForm';
import StatusPanel from './StatusPanel';
import DeployModal from './DeployModal';
import './styles.css';

const GpuConfigPage: React.FC = () => {
  const { isAdmin } = useAuth();
  const { profiles, migFamilies } = useProfiles();
  const { config: currentConfig, fetchConfig } = useCurrentConfig();
  const { status, fetchStatus } = useClusterStatus();

  const [activeTab, setActiveTab] = React.useState<string | number>('builtin');
  const [selectedProfileId, setSelectedProfileId] = React.useState<string | null>(null);
  const [migSlices, setMigSlices] = React.useState<MIGSlice[]>([]);
  const [customConfig, setCustomConfig] = React.useState<CustomConfig>({
    gpuProduct: '', gpuCount: 8, gpuMemory: 49152, migEnabled: false,
  });
  const [deployRequest, setDeployRequest] = React.useState<DeployRequest | null>(null);
  const [modalOpen, setModalOpen] = React.useState(false);

  const selectedProfile = profiles.find((p) => p.id === selectedProfileId) || null;
  const selectedMigFamily = selectedProfile?.migFamily
    ? migFamilies.find((f) => f.id === selectedProfile.migFamily) || null
    : null;

  React.useEffect(() => {
    if (selectedProfile?.migSlices) {
      setMigSlices([...selectedProfile.migSlices]);
    } else {
      setMigSlices([]);
    }
  }, [selectedProfileId]);

  const handleDeploy = () => {
    let req: DeployRequest;
    if (activeTab === 'builtin' && selectedProfile) {
      req = {
        profile: selectedProfile.id,
        gpuProduct: selectedProfile.product,
        gpuCount: selectedProfile.gpuCount,
        gpuMemory: selectedProfile.gpuMemoryMB,
        migStrategy: selectedProfile.migSupport ? 'mixed' : 'none',
        migSlices: migSlices.filter((s) => s.count > 0),
      };
    } else {
      req = {
        profile: 'custom',
        gpuProduct: customConfig.gpuProduct,
        gpuCount: customConfig.gpuCount,
        gpuMemory: customConfig.gpuMemory,
        migStrategy: customConfig.migEnabled ? 'mixed' : 'none',
        migSlices: [],
      };
    }
    setDeployRequest(req);
    setModalOpen(true);
  };

  const handleRefresh = () => {
    fetchConfig();
    fetchStatus();
  };

  const canDeploy = isAdmin && (
    (activeTab === 'builtin' && selectedProfileId) ||
    (activeTab === 'custom' && customConfig.gpuProduct)
  );

  return (
    <>
      <Helmet><title>GPU Configuration</title></Helmet>
        <PageSection variant="default">
          <Toolbar>
            <ToolbarContent>
              <ToolbarItem>
                <Title headingLevel="h1">GPU Profile Configuration</Title>
              </ToolbarItem>
              <ToolbarItem align={{ default: 'alignEnd' }}>
                <Button variant="plain" onClick={handleRefresh} aria-label="Refresh">
                  <SyncIcon />
                </Button>
                <Button
                  variant="primary"
                  onClick={handleDeploy}
                  isDisabled={!canDeploy}
                  style={{ marginLeft: '0.5rem' }}
                >
                  Deploy
                </Button>
              </ToolbarItem>
            </ToolbarContent>
          </Toolbar>
        </PageSection>

        <PageSection>
          <Tabs activeKey={activeTab} onSelect={(_e, key) => setActiveTab(key)}>
            <Tab eventKey="builtin" title={<TabTitleText>Builtin Profiles</TabTitleText>}>
              <Gallery hasGutter style={{ marginTop: '1rem' }} className="gpu-profile-gallery">
                {profiles.map((p) => (
                  <ProfileCard
                    key={p.id}
                    profile={p}
                    isActive={currentConfig?.profile === p.id}
                    isSelected={selectedProfileId === p.id}
                    isAdmin={isAdmin}
                    onSelect={() => setSelectedProfileId(p.id)}
                  />
                ))}
              </Gallery>
            </Tab>
            <Tab eventKey="custom" title={<TabTitleText>Custom</TabTitleText>}>
              <div style={{ marginTop: '1rem', maxWidth: '600px' }}>
                <CustomProfileForm config={customConfig} onChange={setCustomConfig} />
              </div>
            </Tab>
          </Tabs>
        </PageSection>

        {activeTab === 'builtin' && selectedProfile?.migSupport && (
          <PageSection>
            <MigConfigPanel
              migFamily={selectedMigFamily}
              slices={migSlices}
              onChange={setMigSlices}
            />
          </PageSection>
        )}

        <PageSection>
          <StatusPanel status={status} currentConfig={currentConfig} />
        </PageSection>
      <DeployModal
        isOpen={modalOpen}
        request={deployRequest}
        onClose={() => setModalOpen(false)}
        onComplete={() => {
          fetchConfig();
          fetchStatus();
        }}
      />
    </>
  );
};

export default GpuConfigPage;

import * as React from 'react';
import {
  Card, CardHeader, CardTitle, CardBody,
  DescriptionList, DescriptionListGroup, DescriptionListTerm, DescriptionListDescription,
  Label,
} from '@patternfly/react-core';
import { CheckCircleIcon } from '@patternfly/react-icons';
import { GPUProfile } from '../utils/constants';

interface Props {
  profile: GPUProfile;
  isActive: boolean;
  isSelected: boolean;
  isAdmin: boolean;
  onSelect: () => void;
}

const ProfileCard: React.FC<Props> = ({ profile, isActive, isSelected, isAdmin, onSelect }) => {
  const migGpuCount = profile.migSlices && profile.migSlices.length > 0 ? profile.gpuCount : 0;

  return (
    <Card
      onClick={onSelect}
      className={`gpu-profile-card${isSelected ? ' gpu-profile-card--selected' : ''}`}
    >
      <CardHeader
        actions={{
          actions: isActive ? (
            <Label color="green" icon={<CheckCircleIcon />}>Active</Label>
          ) : null,
        }}
      >
        <CardTitle>{profile.name}</CardTitle>
      </CardHeader>
      <CardBody>
        <DescriptionList isHorizontal isCompact>
          <DescriptionListGroup>
            <DescriptionListTerm>Architecture</DescriptionListTerm>
            <DescriptionListDescription>{profile.architecture}</DescriptionListDescription>
          </DescriptionListGroup>
          <DescriptionListGroup>
            <DescriptionListTerm>Memory</DescriptionListTerm>
            <DescriptionListDescription>{profile.memory}</DescriptionListDescription>
          </DescriptionListGroup>
          <DescriptionListGroup>
            <DescriptionListTerm>Full GPUs</DescriptionListTerm>
            <DescriptionListDescription>{profile.gpuCount}</DescriptionListDescription>
          </DescriptionListGroup>
          {migGpuCount > 0 && (
            <DescriptionListGroup>
              <DescriptionListTerm>MIG GPUs</DescriptionListTerm>
              <DescriptionListDescription>{migGpuCount}</DescriptionListDescription>
            </DescriptionListGroup>
          )}
          <DescriptionListGroup>
            <DescriptionListTerm>MIG Support</DescriptionListTerm>
            <DescriptionListDescription>
              {profile.migSupport ? (
                <Label color="blue">Yes</Label>
              ) : (
                <Label color="grey">No</Label>
              )}
            </DescriptionListDescription>
          </DescriptionListGroup>
          {profile.migSlices && profile.migSlices.length > 0 && (
            <DescriptionListGroup>
              <DescriptionListTerm>Default MIG slices</DescriptionListTerm>
              <DescriptionListDescription>
                {profile.migSlices.map((s) => (
                  <div key={s.name}>
                    {s.name.replace('nvidia.com/mig-', '')}: {s.count}
                  </div>
                ))}
              </DescriptionListDescription>
            </DescriptionListGroup>
          )}
        </DescriptionList>
      </CardBody>
    </Card>
  );
};

export default ProfileCard;

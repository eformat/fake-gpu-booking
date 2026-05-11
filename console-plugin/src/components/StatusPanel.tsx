import * as React from 'react';
import {
  Card, CardTitle, CardBody,
  DescriptionList, DescriptionListGroup, DescriptionListTerm, DescriptionListDescription,
  Label, Spinner,
} from '@patternfly/react-core';
import { CheckCircleIcon, ExclamationCircleIcon } from '@patternfly/react-icons';
import { ClusterStatus, DeployRequest } from '../utils/constants';

interface Props {
  status: ClusterStatus | null;
  currentConfig: DeployRequest | null;
}

const StatusPanel: React.FC<Props> = ({ status, currentConfig }) => {
  if (!status) {
    return (
      <Card>
        <CardTitle>Cluster Status</CardTitle>
        <CardBody><Spinner size="md" /> Loading...</CardBody>
      </Card>
    );
  }

  const readyNodes = status.nodes?.filter((n) => n.ready).length || 0;
  const totalNodes = status.nodes?.length || 0;

  return (
    <Card>
      <CardTitle>Cluster Status</CardTitle>
      <CardBody>
        <DescriptionList isHorizontal isCompact>
          <DescriptionListGroup>
            <DescriptionListTerm>Active Profile</DescriptionListTerm>
            <DescriptionListDescription>
              {currentConfig ? (
                <Label color="blue">{currentConfig.profile} ({currentConfig.gpuProduct})</Label>
              ) : (
                <Label color="grey">None configured</Label>
              )}
            </DescriptionListDescription>
          </DescriptionListGroup>
          <DescriptionListGroup>
            <DescriptionListTerm>Worker Nodes</DescriptionListTerm>
            <DescriptionListDescription>{readyNodes} Ready / {totalNodes} Total</DescriptionListDescription>
          </DescriptionListGroup>
          {status.deployments && (
            <DescriptionListGroup>
              <DescriptionListTerm>Deployments</DescriptionListTerm>
              <DescriptionListDescription>
                {status.deployments.filter((d) => d.desired > 0).map((d) => (
                  <div key={d.name}>
                    {d.ready === d.desired ? (
                      <CheckCircleIcon color="green" />
                    ) : (
                      <ExclamationCircleIcon color="orange" />
                    )}{' '}
                    {d.name} ({d.ready}/{d.desired})
                  </div>
                ))}
              </DescriptionListDescription>
            </DescriptionListGroup>
          )}
          {status.daemonSets && (
            <DescriptionListGroup>
              <DescriptionListTerm>DaemonSets</DescriptionListTerm>
              <DescriptionListDescription>
                {status.daemonSets.map((ds) => (
                  <div key={ds.name}>
                    {ds.ready === ds.desired ? (
                      <CheckCircleIcon color="green" />
                    ) : (
                      <ExclamationCircleIcon color="orange" />
                    )}{' '}
                    {ds.name} ({ds.ready}/{ds.desired})
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

export default StatusPanel;

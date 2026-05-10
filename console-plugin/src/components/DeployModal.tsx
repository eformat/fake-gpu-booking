import * as React from 'react';
import {
  Modal, ModalVariant, ModalBody, ModalFooter, ModalHeader,
  Button, Alert, ActionGroup,
  DescriptionList, DescriptionListGroup, DescriptionListTerm, DescriptionListDescription,
  Label,
} from '@patternfly/react-core';
import { CheckCircleIcon, ExclamationCircleIcon } from '@patternfly/react-icons';
import { DeployRequest, DeployResult } from '../utils/constants';
import { deployProfile } from '../utils/api';

interface Props {
  isOpen: boolean;
  request: DeployRequest | null;
  onClose: () => void;
  onComplete: () => void;
}

const DeployModal: React.FC<Props> = ({ isOpen, request, onClose, onComplete }) => {
  const [deploying, setDeploying] = React.useState(false);
  const [result, setResult] = React.useState<DeployResult | null>(null);
  const [error, setError] = React.useState<string | null>(null);

  const handleDeploy = async () => {
    if (!request) return;
    setDeploying(true);
    setError(null);
    setResult(null);
    try {
      const res = await deployProfile(request);
      setResult(res);
      onComplete();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Deploy failed');
    } finally {
      setDeploying(false);
    }
  };

  const handleClose = () => {
    setResult(null);
    setError(null);
    setDeploying(false);
    onClose();
  };

  if (!request) return null;

  const migSliceStr = request.migSlices?.filter((s) => s.count > 0)
    .map((s) => `${s.name.replace('nvidia.com/mig-', '')}: ${s.count}`)
    .join(', ');

  return (
    <Modal
      variant={ModalVariant.medium}
      isOpen={isOpen}
      onClose={handleClose}
    >
      <ModalHeader title={result ? 'Deploy Result' : 'Deploy GPU Configuration'} />
      <ModalBody>
        {!result && !error && (
          <>
            <p style={{ marginBottom: '1rem' }}>
              Deploy the following configuration to all worker nodes?
            </p>
            <DescriptionList isHorizontal isCompact>
              <DescriptionListGroup>
                <DescriptionListTerm>Profile</DescriptionListTerm>
                <DescriptionListDescription>{request.profile}</DescriptionListDescription>
              </DescriptionListGroup>
              <DescriptionListGroup>
                <DescriptionListTerm>GPU Product</DescriptionListTerm>
                <DescriptionListDescription>{request.gpuProduct}</DescriptionListDescription>
              </DescriptionListGroup>
              <DescriptionListGroup>
                <DescriptionListTerm>GPU Count</DescriptionListTerm>
                <DescriptionListDescription>{request.gpuCount}</DescriptionListDescription>
              </DescriptionListGroup>
              <DescriptionListGroup>
                <DescriptionListTerm>MIG Strategy</DescriptionListTerm>
                <DescriptionListDescription>{request.migStrategy || 'none'}</DescriptionListDescription>
              </DescriptionListGroup>
              {migSliceStr && (
                <DescriptionListGroup>
                  <DescriptionListTerm>MIG Slices</DescriptionListTerm>
                  <DescriptionListDescription>{migSliceStr}</DescriptionListDescription>
                </DescriptionListGroup>
              )}
            </DescriptionList>
            <Alert variant="warning" isInline title="This will update the topology ConfigMap, label nodes, and restart GPU operator workloads." style={{ marginTop: '1rem' }} />
          </>
        )}

        {error && <Alert variant="danger" isInline title={error} />}

        {result && (
          <div>
            {result.steps.map((step) => (
              <div key={step.step} style={{ marginBottom: '0.5rem' }}>
                {step.status === 'ok' ? (
                  <Label color="green" icon={<CheckCircleIcon />}>
                    {step.step}{step.message ? `: ${step.message}` : ''}
                  </Label>
                ) : (
                  <Label color="red" icon={<ExclamationCircleIcon />}>
                    {step.step}: {step.message}
                  </Label>
                )}
              </div>
            ))}
          </div>
        )}
      </ModalBody>
      <ModalFooter>
        <ActionGroup>
          {result ? (
            <Button onClick={handleClose}>Close</Button>
          ) : (
            <>
              <Button variant="primary" onClick={handleDeploy} isDisabled={deploying} isLoading={deploying}>
                {deploying ? 'Deploying...' : 'Deploy'}
              </Button>
              <Button variant="link" onClick={handleClose} isDisabled={deploying}>
                Cancel
              </Button>
            </>
          )}
        </ActionGroup>
      </ModalFooter>
    </Modal>
  );
};

export default DeployModal;

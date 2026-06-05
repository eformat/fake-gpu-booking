import * as React from 'react';
import {
  Card, CardTitle, CardBody,
  DataList, DataListItem, DataListItemRow, DataListItemCells, DataListCell, DataListCheck,
  Label,
} from '@patternfly/react-core';
import { CheckCircleIcon, ExclamationCircleIcon } from '@patternfly/react-icons';
import { NodeStatus } from '../utils/constants';

interface Props {
  nodes: NodeStatus[];
  selectedNodes: string[];
  onChange: (nodes: string[]) => void;
}

const NodePicker: React.FC<Props> = ({ nodes, selectedNodes, onChange }) => {
  const toggle = (name: string) => {
    if (selectedNodes.includes(name)) {
      onChange(selectedNodes.filter((n) => n !== name));
    } else {
      onChange([...selectedNodes, name]);
    }
  };

  return (
    <Card>
      <CardTitle>GPU Target Nodes</CardTitle>
      <CardBody>
        <p style={{ marginBottom: '0.5rem' }}>Select which worker nodes should receive simulated GPU resources.</p>
        <DataList aria-label="GPU target node selection" isCompact>
          {nodes.map((node) => (
            <DataListItem key={node.name} id={`node-${node.name}`}>
              <DataListItemRow>
                <DataListCheck
                  id={`node-check-${node.name}`}
                  aria-labelledby={`node-label-${node.name}`}
                  isChecked={selectedNodes.includes(node.name)}
                  onChange={(_event, _checked) => toggle(node.name)}
                />
                <DataListItemCells
                  dataListCells={[
                    <DataListCell key="name" id={`node-label-${node.name}`}>
                      <strong>{node.name}</strong>
                    </DataListCell>,
                    <DataListCell key="ready">
                      {node.ready ? (
                        <Label color="green" icon={<CheckCircleIcon />} isCompact>Ready</Label>
                      ) : (
                        <Label color="red" icon={<ExclamationCircleIcon />} isCompact>Not Ready</Label>
                      )}
                    </DataListCell>,
                    <DataListCell key="gpu">
                      {node.gpuPool ? (
                        <Label color="blue" isCompact>GPU: {node.gpuPool}</Label>
                      ) : (
                        <Label color="grey" isCompact>No GPU</Label>
                      )}
                    </DataListCell>,
                  ]}
                />
              </DataListItemRow>
            </DataListItem>
          ))}
        </DataList>
      </CardBody>
    </Card>
  );
};

export default NodePicker;

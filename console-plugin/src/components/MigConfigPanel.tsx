import * as React from 'react';
import {
  Card, CardTitle, CardBody,
  FormGroup, NumberInput,
  HelperText, HelperTextItem,
} from '@patternfly/react-core';
import { MIGSlice, MIGFamily } from '../utils/constants';

interface Props {
  migFamily: MIGFamily | null;
  slices: MIGSlice[];
  onChange: (slices: MIGSlice[]) => void;
}

const MigConfigPanel: React.FC<Props> = ({ migFamily, slices, onChange }) => {
  if (!migFamily) return null;

  const handleCountChange = (name: string, value: number) => {
    const updated = slices.map((s) =>
      s.name === name ? { ...s, count: Math.max(0, value) } : s
    );
    onChange(updated);
  };

  return (
    <Card>
      <CardTitle>MIG Configuration ({migFamily.match})</CardTitle>
      <CardBody>
        <HelperText>
          <HelperTextItem>Configure MIG slice counts per node. Set to 0 to disable a slice type.</HelperTextItem>
        </HelperText>
        {slices.map((slice) => (
          <FormGroup
            key={slice.name}
            label={slice.name.replace('nvidia.com/mig-', '')}
            fieldId={`mig-${slice.name}`}
            style={{ marginTop: '0.5rem' }}
          >
            <NumberInput
              id={`mig-${slice.name}`}
              value={slice.count}
              min={0}
              onMinus={() => handleCountChange(slice.name, slice.count - 1)}
              onPlus={() => handleCountChange(slice.name, slice.count + 1)}
              onChange={(event) => {
                const val = parseInt((event.target as HTMLInputElement).value, 10);
                if (!isNaN(val)) handleCountChange(slice.name, val);
              }}
            />
          </FormGroup>
        ))}
      </CardBody>
    </Card>
  );
};

export default MigConfigPanel;

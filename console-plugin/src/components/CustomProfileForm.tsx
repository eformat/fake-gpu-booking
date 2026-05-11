import * as React from 'react';
import {
  Card, CardTitle, CardBody,
  FormGroup, NumberInput, Switch,
  Select, SelectOption, SelectList, MenuToggle, TextInputGroup, TextInputGroupMain,
} from '@patternfly/react-core';

export interface CustomConfig {
  gpuProduct: string;
  gpuCount: number;
  gpuMemory: number;
  migEnabled: boolean;
}

interface Props {
  config: CustomConfig;
  migAvailable: boolean;
  onChange: (config: CustomConfig) => void;
}

const COMMON_PRODUCTS = [
  'NVIDIA-A100-SXM4-40GB',
  'NVIDIA-A100-SXM4-80GB',
  'NVIDIA-A100-PCIE-40GB',
  'NVIDIA-H100-80GB-HBM3',
  'NVIDIA-H100-NVL',
  'NVIDIA-H200',
  'NVIDIA-B200',
  'NVIDIA-GB200-NVL',
  'NVIDIA-L40S',
  'NVIDIA-L40',
  'NVIDIA-L4',
  'NVIDIA-T4',
  'NVIDIA-V100-SXM2-16GB',
  'NVIDIA-V100-SXM2-32GB',
  'NVIDIA-A10G',
  'NVIDIA-A30',
];

const CustomProfileForm: React.FC<Props> = ({ config, migAvailable, onChange }) => {
  const [isOpen, setIsOpen] = React.useState(false);
  const [filterValue, setFilterValue] = React.useState('');

  const filtered = filterValue
    ? COMMON_PRODUCTS.filter((p) => p.toLowerCase().includes(filterValue.toLowerCase()))
    : COMMON_PRODUCTS;

  const handleSelect = (_event: any, value: string) => {
    onChange({ ...config, gpuProduct: value });
    setFilterValue('');
    setIsOpen(false);
  };

  const handleTextChange = (_event: any, value: string) => {
    setFilterValue(value);
    onChange({ ...config, gpuProduct: value });
    if (!isOpen) setIsOpen(true);
  };

  const toggle = (toggleRef: React.Ref<any>) => (
    <MenuToggle ref={toggleRef} variant="typeahead" onClick={() => setIsOpen(!isOpen)} isExpanded={isOpen} isFullWidth>
      <TextInputGroup isPlain>
        <TextInputGroupMain
          value={filterValue || config.gpuProduct}
          onClick={() => setIsOpen(true)}
          onChange={handleTextChange}
          placeholder="Select or type a GPU product name"
          autoComplete="off"
        />
      </TextInputGroup>
    </MenuToggle>
  );

  return (
    <Card>
      <CardTitle>Custom GPU Profile</CardTitle>
      <CardBody>
        <FormGroup label="GPU Product Name" fieldId="gpu-product" isRequired>
          <Select
            id="gpu-product"
            isOpen={isOpen}
            selected={config.gpuProduct}
            onSelect={handleSelect}
            onOpenChange={setIsOpen}
            toggle={toggle}
          >
            <SelectList>
              {filtered.map((product) => (
                <SelectOption key={product} value={product}>
                  {product}
                </SelectOption>
              ))}
            </SelectList>
          </Select>
        </FormGroup>
        <FormGroup label="GPU Count per Node" fieldId="gpu-count" style={{ marginTop: '0.5rem' }}>
          <NumberInput
            id="gpu-count"
            value={config.gpuCount}
            min={1}
            max={16}
            onMinus={() => onChange({ ...config, gpuCount: Math.max(1, config.gpuCount - 1) })}
            onPlus={() => onChange({ ...config, gpuCount: Math.min(16, config.gpuCount + 1) })}
            onChange={(event) => {
              const val = parseInt((event.target as HTMLInputElement).value, 10);
              if (!isNaN(val) && val > 0) onChange({ ...config, gpuCount: val });
            }}
          />
        </FormGroup>
        <FormGroup label="GPU Memory (MB)" fieldId="gpu-memory" style={{ marginTop: '0.5rem' }}>
          <NumberInput
            id="gpu-memory"
            value={config.gpuMemory}
            min={1024}
            onMinus={() => onChange({ ...config, gpuMemory: Math.max(1024, config.gpuMemory - 1024) })}
            onPlus={() => onChange({ ...config, gpuMemory: config.gpuMemory + 1024 })}
            onChange={(event) => {
              const val = parseInt((event.target as HTMLInputElement).value, 10);
              if (!isNaN(val) && val > 0) onChange({ ...config, gpuMemory: val });
            }}
          />
        </FormGroup>
        {migAvailable && (
          <FormGroup fieldId="mig-toggle" style={{ marginTop: '0.5rem' }}>
            <Switch
              id="mig-toggle"
              label="MIG Enabled"
              isChecked={config.migEnabled}
              onChange={(_event, checked) => onChange({ ...config, migEnabled: checked })}
            />
          </FormGroup>
        )}
      </CardBody>
    </Card>
  );
};

export default CustomProfileForm;

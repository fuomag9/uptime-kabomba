"use client";

import { useState } from 'react';
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Button } from '@/components/ui/button';

export type PeriodType = '1h' | '3h' | '6h' | '24h' | 'custom';

interface PeriodOption {
  value: PeriodType;
  label: string;
  description: string;
}

interface PeriodSelectorProps {
  value: PeriodType;
  onChange: (period: PeriodType, customRange?: { start: Date; end: Date }) => void;
  customStart?: Date;
  customEnd?: Date;
  className?: string;
  compact?: boolean;
}

const PERIOD_OPTIONS: PeriodOption[] = [
  { value: '1h', label: '1h', description: 'Last hour' },
  { value: '3h', label: '3h', description: 'Last 3 hours' },
  { value: '6h', label: '6h', description: 'Last 6 hours' },
  { value: '24h', label: '24h', description: 'Last 24 hours' },
  { value: 'custom', label: 'Custom', description: 'Custom date range' },
];

const COMPACT_OPTIONS = PERIOD_OPTIONS.filter(o => o.value !== 'custom');

function formatDateForInput(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  const hours = String(date.getHours()).padStart(2, '0');
  const minutes = String(date.getMinutes()).padStart(2, '0');
  return `${year}-${month}-${day}T${hours}:${minutes}`;
}

function formatDateShort(date: Date): string {
  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
}

export default function PeriodSelector({
  value,
  onChange,
  customStart,
  customEnd,
  className = '',
  compact = false,
}: PeriodSelectorProps) {
  const [showCustomPicker, setShowCustomPicker] = useState(false);
  const [tempStart, setTempStart] = useState(customStart || new Date(Date.now() - 7 * 24 * 60 * 60 * 1000));
  const [tempEnd, setTempEnd] = useState(customEnd || new Date());

  const options = compact ? COMPACT_OPTIONS : PERIOD_OPTIONS;

  const handlePeriodClick = (period: PeriodType) => {
    if (period === 'custom') {
      setShowCustomPicker(true);
    } else {
      onChange(period);
      setShowCustomPicker(false);
    }
  };

  const handleCustomApply = () => {
    onChange('custom', { start: tempStart, end: tempEnd });
    setShowCustomPicker(false);
  };

  if (compact) {
    return (
      <ToggleGroup
        type="single"
        value={value}
        onValueChange={(val) => {
          if (val) handlePeriodClick(val as PeriodType);
        }}
        className={className}
        size="sm"
      >
        {options.map((option) => (
          <ToggleGroupItem
            key={option.value}
            value={option.value}
            title={option.description}
          >
            {option.label}
          </ToggleGroupItem>
        ))}
      </ToggleGroup>
    );
  }

  return (
    <div className={`relative ${className}`}>
      <ToggleGroup
        type="single"
        value={value}
        onValueChange={(val) => {
          if (val) handlePeriodClick(val as PeriodType);
        }}
        variant="outline"
      >
        {options.map((option) => (
          <ToggleGroupItem
            key={option.value}
            value={option.value}
            title={option.description}
          >
            {option.label}
          </ToggleGroupItem>
        ))}
      </ToggleGroup>

      <Dialog open={showCustomPicker} onOpenChange={setShowCustomPicker}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Select Custom Date Range</DialogTitle>
          </DialogHeader>

          <div className="space-y-4">
            <div className="space-y-2">
              <Label>Start Date & Time</Label>
              <Input
                type="datetime-local"
                value={formatDateForInput(tempStart)}
                onChange={(e) => setTempStart(new Date(e.target.value))}
              />
            </div>

            <div className="space-y-2">
              <Label>End Date & Time</Label>
              <Input
                type="datetime-local"
                value={formatDateForInput(tempEnd)}
                onChange={(e) => setTempEnd(new Date(e.target.value))}
              />
            </div>
          </div>

          <div className="flex justify-end gap-3 mt-4">
            <Button
              variant="outline"
              onClick={() => setShowCustomPicker(false)}
            >
              Cancel
            </Button>
            <Button onClick={handleCustomApply}>
              Apply
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {value === 'custom' && customStart && customEnd && (
        <div className="mt-2 text-xs text-muted-foreground">
          {formatDateShort(customStart)} - {formatDateShort(customEnd)}
        </div>
      )}
    </div>
  );
}

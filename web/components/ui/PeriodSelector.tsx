"use client";

import { useState } from 'react';

export type PeriodType = '1h' | '24h' | '7d' | '30d' | '90d' | 'custom';

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
  { value: '24h', label: '24h', description: 'Last 24 hours' },
  { value: '7d', label: '7d', description: 'Last 7 days' },
  { value: '30d', label: '30d', description: 'Last 30 days' },
  { value: '90d', label: '90d', description: 'Last 90 days' },
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
      <div className={`flex gap-1 ${className}`}>
        {options.map((option) => (
          <button
            key={option.value}
            onClick={() => handlePeriodClick(option.value)}
            className={`
              text-xs px-2 py-1 rounded-md font-medium transition-all duration-200
              ${
                value === option.value
                  ? 'bg-blue-600 text-white shadow-sm'
                  : 'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-600'
              }
            `}
            title={option.description}
          >
            {option.label}
          </button>
        ))}
      </div>
    );
  }

  return (
    <div className={`relative ${className}`}>
      {/* Segmented Control */}
      <div className="inline-flex items-center rounded-lg bg-gray-100 dark:bg-gray-800 p-1 shadow-inner">
        {options.map((option) => (
          <button
            key={option.value}
            onClick={() => handlePeriodClick(option.value)}
            className={`
              relative px-4 py-2 text-sm font-medium rounded-md transition-all duration-200
              ${
                value === option.value
                  ? 'bg-white dark:bg-gray-700 text-gray-900 dark:text-white shadow-sm'
                  : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white'
              }
            `}
            title={option.description}
          >
            {option.label}
          </button>
        ))}
      </div>

      {/* Custom Date Range Picker Modal */}
      {showCustomPicker && (
        <div className="fixed inset-0 bg-black/60 dark:bg-black/80 backdrop-blur-sm flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 rounded-xl shadow-2xl p-6 max-w-md w-full mx-4">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
              Select Custom Date Range
            </h3>

            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Start Date & Time
                </label>
                <input
                  type="datetime-local"
                  value={formatDateForInput(tempStart)}
                  onChange={(e) => setTempStart(new Date(e.target.value))}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  End Date & Time
                </label>
                <input
                  type="datetime-local"
                  value={formatDateForInput(tempEnd)}
                  onChange={(e) => setTempEnd(new Date(e.target.value))}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
                />
              </div>
            </div>

            <div className="flex justify-end gap-3 mt-6">
              <button
                onClick={() => setShowCustomPicker(false)}
                className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors font-medium"
              >
                Cancel
              </button>
              <button
                onClick={handleCustomApply}
                className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-500 transition-colors font-medium shadow-sm"
              >
                Apply
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Show selected custom range */}
      {value === 'custom' && customStart && customEnd && (
        <div className="mt-2 text-xs text-gray-500 dark:text-gray-400">
          {formatDateShort(customStart)} - {formatDateShort(customEnd)}
        </div>
      )}
    </div>
  );
}

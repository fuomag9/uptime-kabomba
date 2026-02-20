"use client";

import { Heartbeat } from '@/lib/api';
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts';

interface HeartbeatChartProps {
  heartbeats: Heartbeat[];
  height?: number;
}

const STATUS_LABELS = ['Down', 'Up', 'Pending', 'Maintenance'];
const STATUS_COLORS = ['#ef4444', '#10b981', '#eab308', '#3b82f6'];

function formatTime(timestamp: string): string {
  const date = new Date(timestamp);
  return date.toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });
}

function formatTimeShort(timestamp: string): string {
  const date = new Date(timestamp);
  return date.toLocaleTimeString('en-US', {
    hour: '2-digit',
    minute: '2-digit',
  });
}

interface ChartDataPoint {
  index: number;
  time: number;
  ping: number;
  pingUp: number | null;
  pingDown: number | null;
  pingPending: number | null;
  pingMaintenance: number | null;
  status: number;
  message: string;
  timestamp: string;
}

interface CustomTooltipProps {
  active?: boolean;
  payload?: Array<{ payload: ChartDataPoint }>;
}

const CustomTooltip = ({ active, payload }: CustomTooltipProps) => {
  if (!active || !payload || !payload.length) return null;

  const data = payload[0].payload;

  return (
    <div className="bg-gray-900 dark:bg-black text-white rounded-lg shadow-2xl p-4 border border-gray-700 min-w-[200px]">
      <div className="flex items-center gap-2 mb-3">
        <div
          className="w-3 h-3 rounded-full"
          style={{ backgroundColor: STATUS_COLORS[data.status] }}
        />
        <span className="font-semibold">{STATUS_LABELS[data.status]}</span>
      </div>
      <div className="space-y-2 text-sm">
        <div className="flex justify-between gap-4">
          <span className="text-gray-400">Response:</span>
          <span
            className="font-mono"
            style={{ color: STATUS_COLORS[data.status] ?? '#9ca3af' }}
          >
            {data.ping}ms
          </span>
        </div>
        <div className="flex justify-between gap-4">
          <span className="text-gray-400">Time:</span>
          <span className="text-gray-200">{formatTime(data.timestamp)}</span>
        </div>
        {data.message && (
          <div className="mt-3 pt-3 border-t border-gray-700">
            <div className="text-gray-300 text-xs break-words">{data.message}</div>
          </div>
        )}
      </div>
    </div>
  );
};

function downsampleData(data: ChartDataPoint[], maxPoints: number): ChartDataPoint[] {
  if (data.length <= maxPoints) {
    return data;
  }
  const stride = Math.ceil(data.length / maxPoints);
  const sampled: ChartDataPoint[] = [];
  for (let i = 0; i < data.length; i += stride) {
    sampled.push(data[i]);
  }
  if (sampled[sampled.length - 1]?.index !== data[data.length - 1]?.index) {
    sampled.push(data[data.length - 1]);
  }
  return sampled;
}

export default function HeartbeatChart({ heartbeats, height = 300 }: HeartbeatChartProps) {
  // Transform data for recharts (reverse to show oldest first)
  const chartData: ChartDataPoint[] = heartbeats
    .slice()
    .reverse()
    .map((hb, idx) => ({
      index: idx,
      time: new Date(hb.time).getTime(),
      ping: hb.ping,
      pingUp: hb.status === 1 ? hb.ping : null,
      pingDown: hb.status === 0 ? hb.ping : null,
      pingPending: hb.status === 2 ? hb.ping : null,
      pingMaintenance: hb.status === 3 ? hb.ping : null,
      status: hb.status,
      message: hb.message,
      timestamp: hb.time,
    }));

  let maxPoints = 600;
  if (chartData.length > 1) {
    const firstTime = chartData[0].time;
    const lastTime = chartData[chartData.length - 1].time;
    const rangeHours = Math.abs(lastTime - firstTime) / (1000 * 60 * 60);
    if (rangeHours > 168) {
      maxPoints = 200;
    } else if (rangeHours > 72) {
      maxPoints = 300;
    } else if (rangeHours > 24) {
      maxPoints = 400;
    }
  }

  const displayData = downsampleData(chartData, maxPoints);
  const shouldAnimate = displayData.length <= 400;

  if (chartData.length === 0) {
    return (
      <div
        className="flex items-center justify-center bg-gray-50 dark:bg-gray-800/50 rounded-lg"
        style={{ height }}
      >
        <p className="text-gray-500 dark:text-gray-400">No data available</p>
      </div>
    );
  }

  // Calculate domain for Y axis with padding
  const maxPing = Math.max(...displayData.map(d => d.ping));
  const yDomain = [0, Math.ceil(maxPing * 1.2)];

  return (
    <div className="w-full" style={{ height }}>
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart
          data={displayData}
          margin={{ top: 10, right: 10, left: 0, bottom: 0 }}
        >
          <defs>
            <linearGradient id="colorPingUp" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#10b981" stopOpacity={0.4} />
              <stop offset="95%" stopColor="#10b981" stopOpacity={0} />
            </linearGradient>
            <linearGradient id="colorPingDown" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#ef4444" stopOpacity={0.35} />
              <stop offset="95%" stopColor="#ef4444" stopOpacity={0} />
            </linearGradient>
            <linearGradient id="colorPingPending" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#eab308" stopOpacity={0.35} />
              <stop offset="95%" stopColor="#eab308" stopOpacity={0} />
            </linearGradient>
            <linearGradient id="colorPingMaintenance" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3} />
              <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
            </linearGradient>
          </defs>
          <CartesianGrid
            strokeDasharray="3 3"
            stroke="#374151"
            strokeOpacity={0.5}
            vertical={false}
          />
          <XAxis
            dataKey="index"
            tick={{ fill: '#9ca3af', fontSize: 11 }}
            axisLine={{ stroke: '#4b5563' }}
            tickLine={{ stroke: '#4b5563' }}
            tickFormatter={(value) => {
              const point = displayData[value];
              return point ? formatTimeShort(point.timestamp) : '';
            }}
            interval="preserveStartEnd"
            minTickGap={50}
          />
          <YAxis
            domain={yDomain}
            tick={{ fill: '#9ca3af', fontSize: 11 }}
            axisLine={{ stroke: '#4b5563' }}
            tickLine={{ stroke: '#4b5563' }}
            tickFormatter={(value) => `${value}ms`}
            width={60}
          />
          <Tooltip
            content={<CustomTooltip />}
            cursor={{ stroke: '#6b7280', strokeDasharray: '3 3' }}
          />
          <Area
            type="monotone"
            dataKey="pingDown"
            stroke="#ef4444"
            strokeWidth={2}
            fillOpacity={1}
            fill="url(#colorPingDown)"
            connectNulls={false}
            isAnimationActive={shouldAnimate}
            animationDuration={shouldAnimate ? 500 : 0}
            dot={{
              r: 2,
              fill: '#ef4444',
              stroke: '#ef4444',
            }}
            activeDot={{
              r: 6,
              stroke: '#ef4444',
              strokeWidth: 2,
              fill: '#fff',
            }}
          />
          <Area
            type="monotone"
            dataKey="pingUp"
            stroke="#10b981"
            strokeWidth={2}
            fillOpacity={1}
            fill="url(#colorPingUp)"
            connectNulls={false}
            isAnimationActive={shouldAnimate}
            animationDuration={shouldAnimate ? 500 : 0}
            dot={false}
            activeDot={{
              r: 6,
              stroke: '#10b981',
              strokeWidth: 2,
              fill: '#fff',
            }}
          />
          <Area
            type="monotone"
            dataKey="pingPending"
            stroke="#eab308"
            strokeWidth={2}
            fillOpacity={1}
            fill="url(#colorPingPending)"
            connectNulls={false}
            isAnimationActive={shouldAnimate}
            animationDuration={shouldAnimate ? 500 : 0}
            dot={false}
            activeDot={{
              r: 6,
              stroke: '#eab308',
              strokeWidth: 2,
              fill: '#fff',
            }}
          />
          <Area
            type="monotone"
            dataKey="pingMaintenance"
            stroke="#3b82f6"
            strokeWidth={2}
            fillOpacity={1}
            fill="url(#colorPingMaintenance)"
            connectNulls={false}
            isAnimationActive={shouldAnimate}
            animationDuration={shouldAnimate ? 500 : 0}
            dot={false}
            activeDot={{
              r: 6,
              stroke: '#3b82f6',
              strokeWidth: 2,
              fill: '#fff',
            }}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}

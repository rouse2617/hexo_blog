/**
 * ChartRenderer Component
 * Requirements: 6.1, 6.2, 6.3, 6.4, 6.5
 * 
 * Renders charts using Recharts library
 * Supports line, bar, and pie charts
 * Includes data point hover tooltips
 * Handles chart loading failures
 * Implements data sampling for large datasets (>500 points)
 */

import { useState } from 'react';
import {
  LineChart,
  Line,
  BarChart,
  Bar,
  PieChart,
  Pie,
  Cell,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import type { ChartData } from '../../types/models';
import './ChartRenderer.css';

export interface ChartRendererProps {
  data: ChartData;
}

const COLORS = ['#0088FE', '#00C49F', '#FFBB28', '#FF8042', '#8884D8', '#82CA9D'];
const MAX_DATA_POINTS = 500;

/**
 * Sample data if it exceeds MAX_DATA_POINTS
 * Requirements: 10.5
 */
function sampleData(data: ChartData): ChartData {
  if (data.data.length <= MAX_DATA_POINTS) {
    return data;
  }

  const step = Math.ceil(data.data.length / MAX_DATA_POINTS);
  const sampledData = data.data.filter((_, index) => index % step === 0);

  return {
    ...data,
    data: sampledData,
  };
}

export function ChartRenderer({ data }: ChartRendererProps) {
  const [error, setError] = useState<string | null>(null);

  // Sample data if needed
  const chartData = sampleData(data);

  // Handle chart rendering errors
  const handleError = (err: Error) => {
    console.error('Chart rendering error:', err);
    setError('Failed to render chart');
  };

  // Retry loading chart
  const handleRetry = () => {
    setError(null);
  };

  if (error) {
    return (
      <div className="chart-error" data-testid="chart-error">
        <div className="error-message">{error}</div>
        <button onClick={handleRetry} className="retry-button">
          Retry
        </button>
        <div className="fallback-data">
          <h4>Raw Data:</h4>
          <table>
            <thead>
              <tr>
                <th>X</th>
                <th>Y</th>
                {chartData.data[0]?.label && <th>Label</th>}
              </tr>
            </thead>
            <tbody>
              {chartData.data.map((point, index) => (
                <tr key={index}>
                  <td>{point.x}</td>
                  <td>{point.y}</td>
                  {point.label && <td>{point.label}</td>}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    );
  }

  try {
    switch (chartData.type) {
      case 'line':
        return (
          <div className="chart-container" data-testid="chart-container">
            {chartData.title && <h3 className="chart-title">{chartData.title}</h3>}
            <ResponsiveContainer width="100%" height={300}>
              <LineChart data={chartData.data}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis
                  dataKey="x"
                  label={chartData.xAxisLabel ? { value: chartData.xAxisLabel, position: 'insideBottom', offset: -5 } : undefined}
                />
                <YAxis
                  label={chartData.yAxisLabel ? { value: chartData.yAxisLabel, angle: -90, position: 'insideLeft' } : undefined}
                />
                <Tooltip />
                <Legend />
                <Line type="monotone" dataKey="y" stroke="#8884d8" activeDot={{ r: 8 }} />
              </LineChart>
            </ResponsiveContainer>
          </div>
        );

      case 'bar':
        return (
          <div className="chart-container" data-testid="chart-container">
            {chartData.title && <h3 className="chart-title">{chartData.title}</h3>}
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={chartData.data}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis
                  dataKey="x"
                  label={chartData.xAxisLabel ? { value: chartData.xAxisLabel, position: 'insideBottom', offset: -5 } : undefined}
                />
                <YAxis
                  label={chartData.yAxisLabel ? { value: chartData.yAxisLabel, angle: -90, position: 'insideLeft' } : undefined}
                />
                <Tooltip />
                <Legend />
                <Bar dataKey="y" fill="#82ca9d" />
              </BarChart>
            </ResponsiveContainer>
          </div>
        );

      case 'pie':
        return (
          <div className="chart-container" data-testid="chart-container">
            {chartData.title && <h3 className="chart-title">{chartData.title}</h3>}
            <ResponsiveContainer width="100%" height={300}>
              <PieChart>
                <Pie
                  data={chartData.data}
                  dataKey="y"
                  nameKey="label"
                  cx="50%"
                  cy="50%"
                  outerRadius={80}
                  label
                >
                  {chartData.data.map((_, index) => (
                    <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip />
                <Legend />
              </PieChart>
            </ResponsiveContainer>
          </div>
        );

      default:
        throw new Error(`Unsupported chart type: ${chartData.type}`);
    }
  } catch (err) {
    handleError(err as Error);
    return null;
  }
}

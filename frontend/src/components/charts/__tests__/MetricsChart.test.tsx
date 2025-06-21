import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import MetricsChart, { MetricDataPoint } from '../MetricsChart';

// Mock recharts components
jest.mock('recharts', () => ({
  ResponsiveContainer: ({ children }: any) => <div data-testid="responsive-container">{children}</div>,
  LineChart: ({ children }: any) => <div data-testid="line-chart">{children}</div>,
  AreaChart: ({ children }: any) => <div data-testid="area-chart">{children}</div>,
  BarChart: ({ children }: any) => <div data-testid="bar-chart">{children}</div>,
  Line: () => <div data-testid="line" />,
  Area: () => <div data-testid="area" />,
  Bar: () => <div data-testid="bar" />,
  XAxis: () => <div data-testid="x-axis" />,
  YAxis: () => <div data-testid="y-axis" />,
  CartesianGrid: () => <div data-testid="cartesian-grid" />,
  Tooltip: () => <div data-testid="tooltip" />,
  Legend: () => <div data-testid="legend" />,
}));

const mockData: MetricDataPoint[] = [
  { timestamp: '2024-01-01T10:00:00Z', value: 10 },
  { timestamp: '2024-01-01T10:01:00Z', value: 15 },
  { timestamp: '2024-01-01T10:02:00Z', value: 12 },
  { timestamp: '2024-01-01T10:03:00Z', value: 18 },
  { timestamp: '2024-01-01T10:04:00Z', value: 14 },
];

describe('MetricsChart', () => {
  const defaultProps = {
    data: mockData,
    type: 'line' as const,
    title: 'Test Metric Chart',
    metricName: 'test_metric',
  };

  it('renders chart with title and data', () => {
    render(<MetricsChart {...defaultProps} />);
    
    expect(screen.getByText('Test Metric Chart')).toBeInTheDocument();
    expect(screen.getByTestId('responsive-container')).toBeInTheDocument();
    expect(screen.getByTestId('line-chart')).toBeInTheDocument();
  });

  it('renders loading state', () => {
    render(<MetricsChart {...defaultProps} loading={true} />);
    
    expect(screen.getByText('Loading...')).toBeInTheDocument();
    expect(screen.getByText('Loading chart data...')).toBeInTheDocument();
  });

  it('renders error state', () => {
    const errorMessage = 'Failed to load data';
    render(<MetricsChart {...defaultProps} error={errorMessage} />);
    
    expect(screen.getByText('Failed to load chart')).toBeInTheDocument();
    expect(screen.getByText(errorMessage)).toBeInTheDocument();
  });

  it('renders no data state', () => {
    render(<MetricsChart {...defaultProps} data={[]} />);
    
    expect(screen.getByText('No data available')).toBeInTheDocument();
  });

  it('renders different chart types', () => {
    // Test line chart
    const { rerender } = render(<MetricsChart {...defaultProps} type="line" />);
    expect(screen.getByTestId('line-chart')).toBeInTheDocument();

    // Test area chart
    rerender(<MetricsChart {...defaultProps} type="area" />);
    expect(screen.getByTestId('area-chart')).toBeInTheDocument();

    // Test bar chart
    rerender(<MetricsChart {...defaultProps} type="bar" />);
    expect(screen.getByTestId('bar-chart')).toBeInTheDocument();
  });

  it('displays current value when available', () => {
    render(<MetricsChart {...defaultProps} />);
    
    // Should display the latest value (14 from the last data point)
    expect(screen.getByText('14.0')).toBeInTheDocument();
    expect(screen.getByText('current (avg)')).toBeInTheDocument();
  });

  it('shows real-time indicator', () => {
    render(<MetricsChart {...defaultProps} realTime={true} />);
    
    expect(screen.getByText('Live')).toBeInTheDocument();
  });

  it('displays data point count', () => {
    render(<MetricsChart {...defaultProps} />);
    
    expect(screen.getByText('5 points')).toBeInTheDocument();
  });

  it('handles different units', () => {
    render(<MetricsChart {...defaultProps} unit="MB" />);
    
    // The latest value should be displayed with the unit
    expect(screen.getByText('14.0 MB')).toBeInTheDocument();
  });

  it('handles different aggregation types', () => {
    render(<MetricsChart {...defaultProps} aggregation="max" />);
    
    expect(screen.getByText('current (max)')).toBeInTheDocument();
  });

  it('applies custom color', () => {
    const customColor = '#ff0000';
    render(<MetricsChart {...defaultProps} color={customColor} />);
    
    // Check if the color is applied to the current value display
    const currentValueElement = screen.getByText('14.0');
    expect(currentValueElement).toHaveStyle({ color: customColor });
  });

  it('handles custom height', () => {
    const customHeight = 500;
    render(<MetricsChart {...defaultProps} height={customHeight} />);
    
    const chartContainer = screen.getByTestId('responsive-container').parentElement;
    expect(chartContainer).toHaveStyle({ height: `${customHeight}px` });
  });
});

describe('MetricsChart Value Formatting', () => {
  const createMockData = (value: number): MetricDataPoint[] => [
    { timestamp: '2024-01-01T10:00:00Z', value }
  ];

  it('formats large numbers correctly', () => {
    // Test millions
    render(
      <MetricsChart
        data={createMockData(1500000)}
        type="line"
        title="Large Numbers"
        metricName="test"
      />
    );
    expect(screen.getByText('1.5M')).toBeInTheDocument();

    // Test thousands
    render(
      <MetricsChart
        data={createMockData(1500)}
        type="line"
        title="Thousands"
        metricName="test"
      />
    );
    expect(screen.getByText('1.5K')).toBeInTheDocument();
  });

  it('formats small numbers correctly', () => {
    render(
      <MetricsChart
        data={createMockData(0.123)}
        type="line"
        title="Small Numbers"
        metricName="test"
      />
    );
    expect(screen.getByText('0.123')).toBeInTheDocument();
  });

  it('formats regular numbers correctly', () => {
    render(
      <MetricsChart
        data={createMockData(42.7)}
        type="line"
        title="Regular Numbers"
        metricName="test"
      />
    );
    expect(screen.getByText('42.7')).toBeInTheDocument();
  });
});

describe('MetricsChart Time Formatting', () => {
  it('processes timestamp data correctly', () => {
    const dataWithTimestamps = [
      { timestamp: '2024-01-01T10:00:00Z', value: 10 },
      { timestamp: '2024-01-01T10:30:00Z', value: 20 },
    ];

    render(
      <MetricsChart
        data={dataWithTimestamps}
        type="line"
        title="Time Test"
        metricName="test"
      />
    );

    // The chart should render without errors with timestamp data
    expect(screen.getByTestId('line-chart')).toBeInTheDocument();
  });
});
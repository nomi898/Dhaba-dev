import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it } from 'vitest';
import { mockFetch } from '../test/helpers';
import StatsPage from './StatsPage';

const series = Array.from({ length: 16 }, (_, i) => ({ minute: new Date(Date.UTC(2026, 10, 17, 10, i)).toISOString(), count: i % 3 }));

const SUMMARY = {
  totalOrders: 128, totalRevenueCents: 9876500,
  ordersPerMinute: { current: 2, lastMinute: 4, average5m: 2.6, average15m: 1.8, series },
  statusCounts: { NEW: 1, COOKING: 3, READY: 9 },
  topDishes: [
    { menuItemId: 12, name: 'Chicken Biryani', quantity: 40, revenueCents: 2400000 },
    { menuItemId: 17, name: 'Doodh Patti Chai', quantity: 22, revenueCents: 220000 },
  ],
  recentOrders: [{ id: 128, customerName: 'Zara', tableNumber: 3, totalCents: 60000, itemCount: 1, status: 'COOKING', createdAt: '2026-11-17T10:15:00Z' }],
  worker: { alive: true, workerId: 'kw-1', ageSeconds: 4 },
  queue: { pending: 0, delayed: 2, deadLettered: 0 },
  generatedAt: '2026-11-17T10:15:30Z',
};

describe('StatsPage', () => {
  it('renders KPIs, chart, top dishes and recent orders', async () => {
    mockFetch({ 'GET /api/stats': () => [200, { data: SUMMARY }] });
    render(<MemoryRouter><StatsPage /></MemoryRouter>);
    expect(await screen.findByText('128')).toBeInTheDocument();
    expect(screen.getByText('Rs 98,765 revenue')).toBeInTheDocument();
    expect(screen.getByText('2.6')).toBeInTheDocument();
    expect(screen.getByText('Online')).toBeInTheDocument();
    expect(screen.getByRole('img', { name: /Orders per minute/ })).toBeInTheDocument();
    expect(screen.getByText('Chicken Biryani')).toBeInTheDocument();
    expect(screen.getByText('#128')).toBeInTheDocument();
    expect(screen.getByText('Cooking')).toBeInTheDocument();
  });

  it('flags an offline worker', async () => {
    mockFetch({ 'GET /api/stats': () => [200, { data: { ...SUMMARY, worker: { alive: false } } }] });
    render(<MemoryRouter><StatsPage /></MemoryRouter>);
    expect(await screen.findByText('Offline')).toBeInTheDocument();
  });
});

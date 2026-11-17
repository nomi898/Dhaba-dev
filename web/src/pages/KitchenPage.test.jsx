import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it } from 'vitest';
import { mockFetch } from '../test/helpers';
import KitchenPage from './KitchenPage';

const order = (id, status, extra = {}) => ({
  id, status, customerName: `Guest ${id}`, tableNumber: id, notes: '', totalCents: 1000,
  createdAt: '2026-11-17T10:00:00Z', updatedAt: '2026-11-17T10:01:00Z',
  items: [{ menuItemId: 1, name: 'Chicken Tikka', quantity: 2, unitPriceCents: 500, lineTotalCents: 1000 }], ...extra,
});

describe('KitchenPage', () => {
  it('shows orders in lifecycle columns and requests status changes', async () => {
    const calls = mockFetch({
      'GET /api/orders?status=NEW%2CCOOKING&limit=100': () => [200, { data: [order(1, 'NEW', { notes: 'no onions' }), order(2, 'COOKING')] }],
      'GET /api/orders?status=READY&limit=12': () => [200, { data: [order(3, 'READY')] }],
      'PATCH /api/orders/*': () => [202, { data: { orderId: 1, currentStatus: 'NEW', requestedStatus: 'COOKING' } }],
    });
    const user = userEvent.setup();
    render(<MemoryRouter><KitchenPage /></MemoryRouter>);

    const newCol = await screen.findByRole('region', { name: 'New orders' });
    expect(await within(newCol).findByText('#1')).toBeInTheDocument();
    expect(within(newCol).getByText('“no onions”')).toBeInTheDocument();
    expect(within(screen.getByRole('region', { name: 'Cooking' })).getByText('#2')).toBeInTheDocument();
    const readyCol = screen.getByRole('region', { name: 'Ready for pickup' });
    expect(within(readyCol).getByText('#3')).toBeInTheDocument();
    expect(within(readyCol).queryByRole('button')).not.toBeInTheDocument();

    await user.click(within(newCol).getByRole('button', { name: 'Start cooking' }));
    const patch = calls.find((c) => c.method === 'PATCH');
    expect(patch.url).toBe('/api/orders/1/status');
    expect(patch.body).toEqual({ status: 'COOKING' });
    expect(within(newCol).getByRole('button', { name: /Moving to Cooking/ })).toBeDisabled();
  });

  it('reports rejected transitions', async () => {
    mockFetch({
      'GET /api/orders?status=NEW%2CCOOKING&limit=100': () => [200, { data: [order(2, 'COOKING')] }],
      'GET /api/orders?status=READY&limit=12': () => [200, { data: [] }],
      'PATCH /api/orders/*': () => [409, { error: { code: 'invalid_transition', message: 'cannot move order from READY to READY' } }],
    });
    const user = userEvent.setup();
    render(<MemoryRouter><KitchenPage /></MemoryRouter>);
    await user.click(await screen.findByRole('button', { name: 'Mark ready' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('Order #2: cannot move order');
  });
});

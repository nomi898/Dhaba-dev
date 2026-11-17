import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { describe, expect, it } from 'vitest';
import { MENU, mockFetch } from '../test/helpers';
import OrderPage from './OrderPage';

function renderPage() {
  return render(
    <MemoryRouter initialEntries={['/']}>
      <Routes>
        <Route path="/" element={<OrderPage />} />
        <Route path="/orders/:id" element={<p>tracking page</p>} />
      </Routes>
    </MemoryRouter>,
  );
}

describe('OrderPage', () => {
  it('shows the menu by category with prices and availability', async () => {
    mockFetch({ 'GET /api/menu': () => [200, { data: MENU }] });
    renderPage();
    expect(await screen.findByRole('heading', { name: 'Karahi & Curry' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Breads' })).toBeInTheDocument();
    expect(screen.getByText('Rs 1,450')).toBeInTheDocument();
    expect(screen.getByText('Unavailable')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Add Nihari' })).not.toBeInTheDocument();
  });

  it('adds and removes items and updates the total', async () => {
    mockFetch({ 'GET /api/menu': () => [200, { data: MENU }] });
    const user = userEvent.setup();
    renderPage();
    await user.click(await screen.findByRole('button', { name: 'Add Chicken Karahi' }));
    await user.click(screen.getByRole('button', { name: 'Add Garlic Naan' }));
    await user.click(screen.getByRole('button', { name: 'Add one more Garlic Naan' }));
    expect(screen.getByTestId('cart-total')).toHaveTextContent('Rs 1,690');

    await user.click(screen.getByRole('button', { name: 'Remove one Garlic Naan' }));
    expect(screen.getByTestId('cart-total')).toHaveTextContent('Rs 1,570');

    const cart = screen.getByRole('complementary', { name: 'Your order' });
    await user.click(within(cart).getByRole('button', { name: 'Remove Chicken Karahi' }));
    expect(screen.getByTestId('cart-total')).toHaveTextContent('Rs 120');
  });

  it('validates the form, places the order and navigates to tracking', async () => {
    const calls = mockFetch({
      'GET /api/menu': () => [200, { data: MENU }],
      'POST /api/orders': () => [201, { data: { id: 42, status: 'NEW' } }],
    });
    const user = userEvent.setup();
    renderPage();
    await user.click(await screen.findByRole('button', { name: 'Add Garlic Naan' }));
    await user.click(screen.getByRole('button', { name: /Place order/ }));
    expect(screen.getByText('Please enter your name.')).toBeInTheDocument();

    await user.type(screen.getByLabelText('Name'), 'Ali');
    await user.type(screen.getByLabelText(/Table/), '5');
    await user.click(screen.getByRole('button', { name: /Place order/ }));

    expect(await screen.findByText('tracking page')).toBeInTheDocument();
    const post = calls.find((c) => c.method === 'POST');
    expect(post.body).toEqual({ customerName: 'Ali', tableNumber: 5, notes: '', items: [{ menuItemId: 15, quantity: 1 }] });
    expect(JSON.parse(localStorage.getItem('dhaba.cart.v1')).lines).toHaveLength(0);
  });

  it('shows server validation errors', async () => {
    mockFetch({
      'GET /api/menu': () => [200, { data: MENU }],
      'POST /api/orders': () => [422, { error: { code: 'validation_error', message: 'x', fields: { 'menuItem[15]': 'Garlic Naan is currently unavailable' } } }],
    });
    const user = userEvent.setup();
    renderPage();
    await user.click(await screen.findByRole('button', { name: 'Add Garlic Naan' }));
    await user.type(screen.getByLabelText('Name'), 'Ali');
    await user.click(screen.getByRole('button', { name: /Place order/ }));
    expect(await screen.findByRole('alert')).toHaveTextContent('Garlic Naan is currently unavailable');
  });

  it('shows an error when the menu cannot load', async () => {
    mockFetch({ 'GET /api/menu': () => [503, { error: { code: 'internal_error', message: 'menu unavailable' } }] });
    renderPage();
    expect(await screen.findByText('menu unavailable')).toBeInTheDocument();
  });
});

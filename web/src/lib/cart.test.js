import { describe, expect, it } from 'vitest';
import { MAX_QTY, cartReducer, cartTotals, emptyCart, quantityOf, toOrderPayload, validateCheckout } from './cart';

const biryani = { id: 1, name: 'Biryani', priceCents: 60000, isAvailable: true };
const chai = { id: 2, name: 'Chai', priceCents: 10000, isAvailable: true };

describe('cartReducer', () => {
  it('adds, increments, decrements and removes lines', () => {
    let c = cartReducer(emptyCart, { type: 'add', item: biryani });
    c = cartReducer(c, { type: 'add', item: biryani });
    c = cartReducer(c, { type: 'add', item: chai });
    expect(quantityOf(c, 1)).toBe(2);
    expect(cartTotals(c)).toEqual({ items: 3, totalCents: 130000 });

    c = cartReducer(c, { type: 'decrement', id: 1 });
    expect(quantityOf(c, 1)).toBe(1);
    c = cartReducer(c, { type: 'decrement', id: 1 });
    expect(quantityOf(c, 1)).toBe(0);
    expect(c.lines).toHaveLength(1);

    c = cartReducer(c, { type: 'remove', id: 2 });
    expect(c.lines).toHaveLength(0);
  });

  it('ignores unavailable items and caps quantity', () => {
    expect(cartReducer(emptyCart, { type: 'add', item: { ...chai, isAvailable: false } })).toBe(emptyCart);
    let c = emptyCart;
    for (let i = 0; i < MAX_QTY + 5; i++) c = cartReducer(c, { type: 'add', item: chai });
    expect(quantityOf(c, 2)).toBe(MAX_QTY);
  });

  it('syncs with the latest menu', () => {
    let c = cartReducer(emptyCart, { type: 'add', item: biryani });
    c = cartReducer(c, { type: 'add', item: chai });
    c = cartReducer(c, { type: 'sync', items: [{ ...biryani, priceCents: 65000 }, { ...chai, isAvailable: false }] });
    expect(c.lines).toEqual([{ id: 1, name: 'Biryani', priceCents: 65000, quantity: 1 }]);
  });

  it('clears', () => {
    expect(cartReducer({ lines: [{ id: 1 }] }, { type: 'clear' })).toEqual(emptyCart);
  });
});

describe('checkout helpers', () => {
  it('builds the orders-api payload', () => {
    const c = cartReducer(cartReducer(emptyCart, { type: 'add', item: biryani }), { type: 'add', item: chai });
    expect(toOrderPayload(c, { customerName: ' Sara ', tableNumber: '7', notes: ' mild ' })).toEqual({
      customerName: 'Sara', tableNumber: 7, notes: 'mild',
      items: [{ menuItemId: 1, quantity: 1 }, { menuItemId: 2, quantity: 1 }],
    });
    expect(toOrderPayload(c, { customerName: 'A', tableNumber: '' }).tableNumber).toBeNull();
  });

  it('validates name and table', () => {
    expect(validateCheckout({ customerName: '  ', tableNumber: '' })).toHaveProperty('customerName');
    expect(validateCheckout({ customerName: 'Ali', tableNumber: '0' })).toHaveProperty('tableNumber');
    expect(validateCheckout({ customerName: 'Ali', tableNumber: '2.5' })).toHaveProperty('tableNumber');
    expect(validateCheckout({ customerName: 'Ali', tableNumber: '12' })).toEqual({});
  });
});

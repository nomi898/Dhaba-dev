// Pure cart state. Prices are copied from the menu for display only;
// orders-api re-prices every order from the database.

export const MAX_QTY = 20;

export const emptyCart = { lines: [] };

export function cartReducer(state, action) {
  switch (action.type) {
    case 'add': {
      const { item } = action;
      if (!item?.isAvailable) return state;
      const existing = state.lines.find((l) => l.id === item.id);
      if (existing) {
        if (existing.quantity >= MAX_QTY) return state;
        return { lines: state.lines.map((l) => (l.id === item.id ? { ...l, quantity: l.quantity + 1 } : l)) };
      }
      return { lines: [...state.lines, { id: item.id, name: item.name, priceCents: item.priceCents, quantity: 1 }] };
    }
    case 'decrement':
      return {
        lines: state.lines
          .map((l) => (l.id === action.id ? { ...l, quantity: l.quantity - 1 } : l))
          .filter((l) => l.quantity > 0),
      };
    case 'remove':
      return { lines: state.lines.filter((l) => l.id !== action.id) };
    case 'clear':
      return emptyCart;
    case 'sync': {
      // Drop items that disappeared or became unavailable; refresh names/prices.
      const byId = new Map(action.items.map((i) => [i.id, i]));
      return {
        lines: state.lines
          .filter((l) => byId.get(l.id)?.isAvailable)
          .map((l) => ({ ...l, name: byId.get(l.id).name, priceCents: byId.get(l.id).priceCents })),
      };
    }
    default:
      return state;
  }
}

export function cartTotals(cart) {
  return cart.lines.reduce(
    (acc, l) => ({ items: acc.items + l.quantity, totalCents: acc.totalCents + l.priceCents * l.quantity }),
    { items: 0, totalCents: 0 },
  );
}

export function quantityOf(cart, id) {
  return cart.lines.find((l) => l.id === id)?.quantity ?? 0;
}

export function toOrderPayload(cart, { customerName, tableNumber, notes }) {
  const table = String(tableNumber ?? '').trim();
  return {
    customerName: customerName.trim(),
    tableNumber: table === '' ? null : Number(table),
    notes: (notes || '').trim(),
    items: cart.lines.map((l) => ({ menuItemId: l.id, quantity: l.quantity })),
  };
}

export function validateCheckout({ customerName, tableNumber }) {
  const errors = {};
  if (!customerName.trim()) errors.customerName = 'Please enter your name.';
  else if (customerName.trim().length > 80) errors.customerName = 'Name is too long.';
  const table = String(tableNumber ?? '').trim();
  if (table !== '') {
    const n = Number(table);
    if (!Number.isInteger(n) || n < 1 || n > 500) errors.tableNumber = 'Table must be a number from 1 to 500.';
  }
  return errors;
}

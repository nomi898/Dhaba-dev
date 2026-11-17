import { useState } from 'react';
import { cartTotals, validateCheckout } from '../lib/cart';
import { formatPrice } from '../lib/format';

export default function CartPanel({ cart, dispatch, onCheckout, submitting, serverError }) {
  const [form, setForm] = useState({ customerName: '', tableNumber: '', notes: '' });
  const [errors, setErrors] = useState({});
  const { items, totalCents } = cartTotals(cart);

  const update = (field) => (e) => setForm((f) => ({ ...f, [field]: e.target.value }));

  const submit = (e) => {
    e.preventDefault();
    const found = validateCheckout(form);
    setErrors(found);
    if (Object.keys(found).length === 0 && items > 0) onCheckout(form);
  };

  return (
    <aside className="cart" aria-label="Your order">
      <h2>
        Your order {items > 0 && <span className="count">{items}</span>}
      </h2>
      {cart.lines.length === 0 ? (
        <p className="muted">Your cart is empty. Add dishes from the menu.</p>
      ) : (
        <ul className="cart-lines">
          {cart.lines.map((l) => (
            <li key={l.id}>
              <span className="cart-qty">{l.quantity}×</span>
              <span className="cart-name">{l.name}</span>
              <span className="cart-price">{formatPrice(l.priceCents * l.quantity)}</span>
              <button type="button" className="link-btn" onClick={() => dispatch({ type: 'remove', id: l.id })} aria-label={`Remove ${l.name}`}>
                ✕
              </button>
            </li>
          ))}
        </ul>
      )}
      <div className="cart-total">
        <span>Total</span>
        <strong data-testid="cart-total">{formatPrice(totalCents)}</strong>
      </div>
      <form onSubmit={submit} noValidate>
        <label>
          Name
          <input value={form.customerName} onChange={update('customerName')} maxLength={80} autoComplete="name" aria-invalid={!!errors.customerName} />
        </label>
        {errors.customerName && <p className="field-error">{errors.customerName}</p>}
        <label>
          Table <span className="muted">(optional)</span>
          <input value={form.tableNumber} onChange={update('tableNumber')} inputMode="numeric" aria-invalid={!!errors.tableNumber} />
        </label>
        {errors.tableNumber && <p className="field-error">{errors.tableNumber}</p>}
        <label>
          Notes <span className="muted">(optional)</span>
          <textarea value={form.notes} onChange={update('notes')} maxLength={500} rows={2} placeholder="Less spicy, extra raita…" />
        </label>
        {serverError && <p className="field-error" role="alert">{serverError}</p>}
        <button type="submit" className="btn btn-primary btn-block" disabled={items === 0 || submitting}>
          {submitting ? 'Placing order…' : `Place order · ${formatPrice(totalCents)}`}
        </button>
      </form>
    </aside>
  );
}

import { useState } from 'react';
import { Link, useLocation, useNavigate, useParams } from 'react-router-dom';
import { ordersApi } from '../api/services';
import OrderProgress from '../components/OrderProgress';
import { ErrorBanner, LiveIndicator, Spinner, StatusBadge } from '../components/common';
import { myOrders } from '../hooks/useCart';
import { usePolling } from '../hooks/usePolling';
import { formatClock, formatPrice } from '../lib/format';

const MESSAGES = {
  NEW: 'The kitchen has received your order.',
  COOKING: 'Your food is on the fire. Almost there!',
  READY: 'Your order is ready. Please collect it from the counter.',
};

function TrackOrder({ id }) {
  const location = useLocation();
  const { data: order, error, loading, updatedAt, refresh } = usePolling(
    (signal) => ordersApi.get(id, { signal }),
    3000,
    [id],
  );

  if (loading && !order) return <Spinner label="Loading order…" />;
  if (error?.status === 404) {
    return <ErrorBanner error={{ message: `Order #${id} was not found.` }} />;
  }
  return (
    <>
      <ErrorBanner error={error} onRetry={refresh} />
      {order && (
        <div className="card track-card">
          {location.state?.justPlaced && <div className="banner banner-ok">Order placed! Keep this page open to follow it.</div>}
          <div className="track-head">
            <div>
              <p className="muted">Order</p>
              <h1 className="order-id">#{order.id}</h1>
            </div>
            <div className="track-meta">
              <StatusBadge status={order.status} />
              <LiveIndicator updatedAt={updatedAt} error={error} />
            </div>
          </div>
          <OrderProgress status={order.status} />
          <p className="track-message" aria-live="polite">{MESSAGES[order.status]}</p>
          <ul className="receipt">
            {order.items.map((it) => (
              <li key={it.menuItemId}>
                <span>{it.quantity}× {it.name}</span>
                <span>{formatPrice(it.lineTotalCents)}</span>
              </li>
            ))}
            <li className="receipt-total">
              <span>Total</span>
              <span>{formatPrice(order.totalCents)}</span>
            </li>
          </ul>
          <p className="muted small">
            {order.customerName}
            {order.tableNumber ? ` · Table ${order.tableNumber}` : ''} · placed {formatClock(order.createdAt)}
          </p>
        </div>
      )}
    </>
  );
}

export default function TrackPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [lookup, setLookup] = useState('');
  const recent = myOrders();

  const go = (e) => {
    e.preventDefault();
    const n = lookup.trim().replace(/^#/, '');
    if (/^\d+$/.test(n)) navigate(`/orders/${n}`);
  };

  return (
    <div className="narrow">
      {id && /^\d+$/.test(id) ? <TrackOrder id={id} /> : <h1>Track your order</h1>}
      <form className="lookup" onSubmit={go}>
        <label htmlFor="lookup">Find an order</label>
        <div className="row">
          <input id="lookup" value={lookup} onChange={(e) => setLookup(e.target.value)} placeholder="Order number, e.g. 42" inputMode="numeric" />
          <button type="submit" className="btn">Track</button>
        </div>
      </form>
      {recent.length > 0 && (
        <p className="muted small">
          Your recent orders:{' '}
          {recent.map((o, i) => (
            <span key={o}>
              {i > 0 && ', '}
              <Link to={`/orders/${o}`}>#{o}</Link>
            </span>
          ))}
        </p>
      )}
    </div>
  );
}

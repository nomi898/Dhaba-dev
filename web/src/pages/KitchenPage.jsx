import { useState } from 'react';
import { ordersApi } from '../api/services';
import { ErrorBanner, LiveIndicator, Spinner } from '../components/common';
import { usePolling } from '../hooks/usePolling';
import { STATUS_LABELS, nextStatus, timeAgo } from '../lib/format';

const COLUMNS = [
  { status: 'NEW', title: 'New orders', action: 'Start cooking' },
  { status: 'COOKING', title: 'Cooking', action: 'Mark ready' },
  { status: 'READY', title: 'Ready for pickup', action: null },
];

async function loadBoard(signal) {
  const [active, ready] = await Promise.all([
    ordersApi.list({ status: ['NEW', 'COOKING'], limit: 100 }, { signal }),
    ordersApi.list({ status: ['READY'], limit: 12 }, { signal }),
  ]);
  return [...active, ...ready];
}

export default function KitchenPage() {
  const { data, error, loading, updatedAt, refresh } = usePolling(loadBoard, 3000);
  const [pending, setPending] = useState({});
  const [actionError, setActionError] = useState(null);

  const orders = data || [];
  // A pending request is resolved once the worker has applied it.
  const isPending = (o) => pending[o.id] && pending[o.id] !== o.status;

  const advance = async (order) => {
    const target = nextStatus(order.status);
    if (!target) return;
    setActionError(null);
    setPending((p) => ({ ...p, [order.id]: target }));
    try {
      await ordersApi.setStatus(order.id, target);
      setTimeout(refresh, 700);
    } catch (err) {
      setPending((p) => {
        const { [order.id]: _drop, ...rest } = p;
        return rest;
      });
      if (err.status === 409) refresh();
      setActionError({ message: `Order #${order.id}: ${err.message}` });
    }
  };

  return (
    <div>
      <header className="page-head page-head-row">
        <div>
          <h1>Kitchen</h1>
          <p className="muted">Orders update automatically every few seconds.</p>
        </div>
        <LiveIndicator updatedAt={updatedAt} error={error} />
      </header>
      <ErrorBanner error={error} onRetry={refresh} />
      <ErrorBanner error={actionError} />
      {loading && !data && <Spinner label="Loading orders…" />}
      <div className="board">
        {COLUMNS.map((col) => {
          const list = orders
            .filter((o) => o.status === col.status)
            .sort((a, b) => (col.status === 'READY' ? b.updatedAt.localeCompare(a.updatedAt) : a.createdAt.localeCompare(b.createdAt)));
          return (
            <section key={col.status} className={`column column-${col.status.toLowerCase()}`} aria-label={col.title}>
              <h2>
                {col.title} <span className="count">{list.length}</span>
              </h2>
              {list.length === 0 && <p className="muted small empty">No orders</p>}
              {list.map((o) => (
                <article key={o.id} className="ticket" data-testid={`ticket-${o.id}`}>
                  <header>
                    <strong>#{o.id}</strong>
                    <span className="muted small">{timeAgo(o.createdAt)}</span>
                  </header>
                  <p className="ticket-who">
                    {o.customerName}
                    {o.tableNumber ? ` · Table ${o.tableNumber}` : ' · Takeaway'}
                  </p>
                  <ul>
                    {o.items.map((it) => (
                      <li key={it.menuItemId}>
                        <span className="ticket-qty">{it.quantity}×</span> {it.name}
                      </li>
                    ))}
                  </ul>
                  {o.notes && <p className="ticket-notes">“{o.notes}”</p>}
                  {col.action && (
                    <button type="button" className="btn btn-primary btn-block" disabled={isPending(o)} onClick={() => advance(o)}>
                      {isPending(o) ? `Moving to ${STATUS_LABELS[pending[o.id]]}…` : col.action}
                    </button>
                  )}
                </article>
              ))}
            </section>
          );
        })}
      </div>
    </div>
  );
}

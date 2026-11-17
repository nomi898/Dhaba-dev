import { statsApi } from '../api/services';
import OrdersPerMinuteChart from '../components/OrdersPerMinuteChart';
import { ErrorBanner, LiveIndicator, Spinner, StatusBadge } from '../components/common';
import { usePolling } from '../hooks/usePolling';
import { formatPrice, timeAgo } from '../lib/format';

function Tile({ label, value, hint }) {
  return (
    <div className="tile">
      <span className="tile-label">{label}</span>
      <span className="tile-value">{value}</span>
      {hint && <span className="tile-hint">{hint}</span>}
    </div>
  );
}

export default function StatsPage() {
  const { data: s, error, loading, updatedAt, refresh } = usePolling((signal) => statsApi.summary({ signal }), 5000);

  return (
    <div>
      <header className="page-head page-head-row">
        <div>
          <h1>Statistics</h1>
          <p className="muted">Live numbers from the kitchen. Refreshes every 5 seconds.</p>
        </div>
        <LiveIndicator updatedAt={updatedAt} error={error} />
      </header>
      <ErrorBanner error={error} onRetry={refresh} />
      {loading && !s && <Spinner label="Loading statistics…" />}
      {s && (
        <>
          <div className="tiles">
            <Tile label="Total orders" value={s.totalOrders.toLocaleString()} hint={formatPrice(s.totalRevenueCents) + ' revenue'} />
            <Tile label="Orders this minute" value={s.ordersPerMinute.current} hint={`${s.ordersPerMinute.lastMinute} last minute`} />
            <Tile label="Avg per minute" value={s.ordersPerMinute.average5m} hint={`last 5 min · ${s.ordersPerMinute.average15m} over 15 min`} />
            <Tile
              label="In the kitchen"
              value={s.statusCounts.NEW + s.statusCounts.COOKING}
              hint={`${s.statusCounts.NEW} new · ${s.statusCounts.COOKING} cooking · ${s.statusCounts.READY} ready`}
            />
            <div className="tile">
              <span className="tile-label">Kitchen worker</span>
              <span className={`tile-value worker ${s.worker.alive ? 'worker-up' : 'worker-down'}`}>
                <span aria-hidden="true">{s.worker.alive ? '●' : '▲'}</span> {s.worker.alive ? 'Online' : 'Offline'}
              </span>
              <span className="tile-hint">
                {s.worker.alive ? `heartbeat ${s.worker.ageSeconds ?? '?'}s ago` : 'no heartbeat in 30s'} · queue {s.queue.pending}
                {s.queue.deadLettered > 0 ? ` · ${s.queue.deadLettered} failed` : ''}
              </span>
            </div>
          </div>

          <div className="stats-grid">
            <section className="card">
              <h2>Orders per minute</h2>
              <OrdersPerMinuteChart series={s.ordersPerMinute.series} />
            </section>

            <section className="card">
              <h2>Top-selling dishes</h2>
              {s.topDishes.length === 0 ? (
                <p className="muted">No orders yet.</p>
              ) : (
                <ol className="toplist">
                  {s.topDishes.map((d) => (
                    <li key={d.menuItemId}>
                      <div className="toplist-row">
                        <span>{d.name}</span>
                        <span className="muted small">{d.quantity} sold · {formatPrice(d.revenueCents)}</span>
                      </div>
                      <div className="meter" aria-hidden="true">
                        <span style={{ width: `${(d.quantity / s.topDishes[0].quantity) * 100}%` }} />
                      </div>
                    </li>
                  ))}
                </ol>
              )}
            </section>
          </div>

          <section className="card">
            <h2>Recent orders</h2>
            {s.recentOrders.length === 0 ? (
              <p className="muted">No orders yet.</p>
            ) : (
              <div className="table-wrap">
                <table className="table">
                  <thead>
                    <tr><th>Order</th><th>Customer</th><th>Items</th><th className="num">Total</th><th>Status</th><th>Placed</th></tr>
                  </thead>
                  <tbody>
                    {s.recentOrders.map((o) => (
                      <tr key={o.id}>
                        <td>#{o.id}</td>
                        <td>{o.customerName}{o.tableNumber ? ` · T${o.tableNumber}` : ''}</td>
                        <td>{o.itemCount}</td>
                        <td className="num">{formatPrice(o.totalCents)}</td>
                        <td><StatusBadge status={o.status} /></td>
                        <td className="muted">{timeAgo(o.createdAt)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>
        </>
      )}
    </div>
  );
}

import { STATUS_LABELS } from '../lib/format';

export function StatusBadge({ status }) {
  const icon = { NEW: '●', COOKING: '◐', READY: '✓' }[status] || '?';
  return (
    <span className={`badge badge-${(status || 'unknown').toLowerCase()}`}>
      <span aria-hidden="true">{icon}</span> {STATUS_LABELS[status] || status || 'Unknown'}
    </span>
  );
}

export function ErrorBanner({ error, onRetry }) {
  if (!error) return null;
  return (
    <div className="banner banner-error" role="alert">
      <span>{error.message || 'Something went wrong.'}</span>
      {onRetry && (
        <button type="button" className="btn btn-ghost btn-sm" onClick={onRetry}>
          Retry
        </button>
      )}
    </div>
  );
}

export function Spinner({ label = 'Loading…' }) {
  return (
    <div className="spinner" role="status">
      <span className="spinner-dot" aria-hidden="true" />
      {label}
    </div>
  );
}

export function LiveIndicator({ updatedAt, error }) {
  return (
    <span className={`live ${error ? 'live-stale' : ''}`} title={updatedAt ? `Updated ${updatedAt.toLocaleTimeString()}` : ''}>
      <span className="live-dot" aria-hidden="true" />
      {error ? 'Reconnecting…' : 'Live'}
    </span>
  );
}

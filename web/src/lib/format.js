const pkr = new Intl.NumberFormat('en-PK', { maximumFractionDigits: 0 });

/** Prices are stored in paisa (1/100 rupee). */
export function formatPrice(cents) {
  const value = Number.isFinite(cents) ? cents : 0;
  return `Rs ${pkr.format(Math.round(value / 100))}`;
}

export function timeAgo(iso, now = Date.now()) {
  if (!iso) return '';
  const seconds = Math.max(0, Math.round((now - new Date(iso).getTime()) / 1000));
  if (seconds < 10) return 'just now';
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes} min ago`;
  const hours = Math.floor(minutes / 60);
  return `${hours}h ago`;
}

export function formatClock(iso) {
  if (!iso) return '';
  return new Date(iso).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

export const STATUSES = ['NEW', 'COOKING', 'READY'];

export const STATUS_LABELS = {
  NEW: 'Received',
  COOKING: 'Cooking',
  READY: 'Ready',
};

export function nextStatus(status) {
  const i = STATUSES.indexOf(status);
  return i >= 0 && i < STATUSES.length - 1 ? STATUSES[i + 1] : null;
}

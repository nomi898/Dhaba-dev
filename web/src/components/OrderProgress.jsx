import { STATUSES, STATUS_LABELS } from '../lib/format';

export default function OrderProgress({ status }) {
  const current = STATUSES.indexOf(status);
  return (
    <ol className="progress" aria-label="Order progress">
      {STATUSES.map((s, i) => {
        const state = i < current ? 'done' : i === current ? 'current' : 'todo';
        return (
          <li key={s} className={`progress-step progress-${state}`} aria-current={state === 'current' ? 'step' : undefined}>
            <span className="progress-marker" aria-hidden="true">{i < current || status === 'READY' ? '✓' : i + 1}</span>
            <span>{STATUS_LABELS[s]}</span>
          </li>
        );
      })}
    </ol>
  );
}

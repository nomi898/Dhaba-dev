import { useState } from 'react';
import { formatClock } from '../lib/format';

const W = 640;
const H = 200;
const PAD = { top: 12, right: 8, bottom: 24, left: 28 };

function niceMax(v) {
  if (v <= 4) return 4;
  const pow = 10 ** Math.floor(Math.log10(v));
  const step = [1, 2, 2.5, 5, 10].find((s) => s * pow * 4 >= v) * pow;
  return step * 4;
}

/** Single-series bar chart: orders per minute over the last 15 minutes (+ current minute). */
export default function OrdersPerMinuteChart({ series }) {
  const [hover, setHover] = useState(null);
  const max = niceMax(Math.max(0, ...series.map((p) => p.count)));
  const plotW = W - PAD.left - PAD.right;
  const plotH = H - PAD.top - PAD.bottom;
  const slot = plotW / series.length;
  const barW = Math.max(4, slot - 2 - Math.min(10, slot * 0.35));
  const y = (v) => PAD.top + plotH - (v / max) * plotH;
  const ticks = [0, max / 2, max];

  return (
    <figure className="chart">
      <svg viewBox={`0 0 ${W} ${H}`} role="img" aria-label="Orders per minute, last 15 minutes" onMouseLeave={() => setHover(null)}>
        {ticks.map((t) => (
          <g key={t}>
            <line x1={PAD.left} x2={W - PAD.right} y1={y(t)} y2={y(t)} className="chart-grid" />
            <text x={PAD.left - 6} y={y(t) + 4} className="chart-axis" textAnchor="end">{t}</text>
          </g>
        ))}
        {series.map((p, i) => {
          const x = PAD.left + i * slot + (slot - barW) / 2;
          const h = Math.max(p.count > 0 ? 2 : 0, plotH * (p.count / max));
          const r = Math.min(4, barW / 2, h);
          const isCurrent = i === series.length - 1;
          const top = PAD.top + plotH - h;
          const path = h > 0
            ? `M${x},${PAD.top + plotH} V${top + r} Q${x},${top} ${x + r},${top} H${x + barW - r} Q${x + barW},${top} ${x + barW},${top + r} V${PAD.top + plotH} Z`
            : '';
          return (
            <g key={p.minute} onMouseEnter={() => setHover(i)} onFocus={() => setHover(i)} tabIndex={0} aria-label={`${formatClock(p.minute)}: ${p.count} orders`}>
              <rect x={PAD.left + i * slot} y={PAD.top} width={slot} height={plotH} fill="transparent" />
              {path && <path d={path} className={`chart-bar ${isCurrent ? 'chart-bar-partial' : ''} ${hover === i ? 'chart-bar-hover' : ''}`} />}
              {(i % 5 === 0 || isCurrent) && (
                <text x={PAD.left + i * slot + slot / 2} y={H - 6} className="chart-axis" textAnchor="middle">
                  {isCurrent ? 'now' : formatClock(p.minute)}
                </text>
              )}
            </g>
          );
        })}
        <line x1={PAD.left} x2={W - PAD.right} y1={PAD.top + plotH} y2={PAD.top + plotH} className="chart-baseline" />
      </svg>
      {hover !== null && series[hover] && (
        <div className="chart-tip" style={{ left: `${((PAD.left + hover * slot + slot / 2) / W) * 100}%` }} role="tooltip">
          <strong>{series[hover].count}</strong> order{series[hover].count === 1 ? '' : 's'}
          <span className="muted"> · {formatClock(series[hover].minute)}{hover === series.length - 1 ? ' (so far)' : ''}</span>
        </div>
      )}
      <table className="sr-only">
        <caption>Orders per minute</caption>
        <thead><tr><th>Minute</th><th>Orders</th></tr></thead>
        <tbody>
          {series.map((p) => <tr key={p.minute}><td>{formatClock(p.minute)}</td><td>{p.count}</td></tr>)}
        </tbody>
      </table>
    </figure>
  );
}

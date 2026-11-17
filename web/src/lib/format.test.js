import { describe, expect, it } from 'vitest';
import { formatPrice, nextStatus, timeAgo } from './format';

describe('format', () => {
  it('formats paisa as rupees', () => {
    expect(formatPrice(145000)).toBe('Rs 1,450');
    expect(formatPrice(4000)).toBe('Rs 40');
    expect(formatPrice(undefined)).toBe('Rs 0');
  });

  it('describes elapsed time', () => {
    const now = Date.parse('2026-11-17T10:10:00Z');
    expect(timeAgo('2026-11-17T10:09:55Z', now)).toBe('just now');
    expect(timeAgo('2026-11-17T10:09:30Z', now)).toBe('30s ago');
    expect(timeAgo('2026-11-17T10:05:00Z', now)).toBe('5 min ago');
    expect(timeAgo('2026-11-17T07:00:00Z', now)).toBe('3h ago');
  });

  it('knows the lifecycle', () => {
    expect(nextStatus('NEW')).toBe('COOKING');
    expect(nextStatus('COOKING')).toBe('READY');
    expect(nextStatus('READY')).toBeNull();
  });
});

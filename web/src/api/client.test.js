import { describe, expect, it, vi } from 'vitest';
import { ApiError, request } from './client';

describe('request', () => {
  it('unwraps the data envelope and sends JSON', async () => {
    const fetchMock = vi.fn(async () => ({ ok: true, status: 201, text: async () => '{"data":{"id":5}}' }));
    vi.stubGlobal('fetch', fetchMock);
    await expect(request('/api/orders', { method: 'POST', body: { a: 1 } })).resolves.toEqual({ id: 5 });
    const [, opts] = fetchMock.mock.calls[0];
    expect(opts.headers['Content-Type']).toBe('application/json');
    expect(opts.body).toBe('{"a":1}');
  });

  it('turns error envelopes into ApiError', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => ({
      ok: false, status: 422,
      text: async () => '{"error":{"code":"validation_error","message":"bad","fields":{"customerName":"is required"}}}',
    })));
    const err = await request('/api/orders').catch((e) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect(err).toMatchObject({ status: 422, code: 'validation_error', message: 'bad', fields: { customerName: 'is required' } });
  });

  it('handles non-JSON errors and network failures', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => ({ ok: false, status: 502, text: async () => '<html>Bad gateway</html>' })));
    await expect(request('/x')).rejects.toMatchObject({ status: 502, code: 'http_error' });

    vi.stubGlobal('fetch', vi.fn(async () => { throw new TypeError('Failed to fetch'); }));
    await expect(request('/x')).rejects.toMatchObject({ status: 0, code: 'network_error' });
  });
});

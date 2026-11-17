// All services share one response contract: {"data": ...} on success,
// {"error": {"code", "message", "fields"?}} on failure. Paths are relative:
// Nginx (prod) and the Vite proxy (dev) route /api/* to the right service.

export class ApiError extends Error {
  constructor(status, code, message, fields) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.fields = fields || null;
  }
}

export async function request(path, { method = 'GET', body, signal, timeoutMs = 10000 } = {}) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  const onAbort = () => controller.abort();
  signal?.addEventListener('abort', onAbort);

  let response;
  try {
    response = await fetch(path, {
      method,
      headers: body === undefined ? { Accept: 'application/json' } : { Accept: 'application/json', 'Content-Type': 'application/json' },
      body: body === undefined ? undefined : JSON.stringify(body),
      signal: controller.signal,
    });
  } catch (err) {
    if (signal?.aborted) throw err;
    const timedOut = err?.name === 'AbortError';
    throw new ApiError(0, timedOut ? 'timeout' : 'network_error',
      timedOut ? 'The server took too long to respond.' : 'Cannot reach the server. Check your connection.');
  } finally {
    clearTimeout(timer);
    signal?.removeEventListener('abort', onAbort);
  }

  let payload = null;
  const text = await response.text();
  if (text) {
    try {
      payload = JSON.parse(text);
    } catch {
      payload = null;
    }
  }
  if (!response.ok) {
    const err = payload?.error;
    throw new ApiError(response.status, err?.code || 'http_error',
      err?.message || `Request failed (${response.status})`, err?.fields);
  }
  return payload?.data ?? null;
}

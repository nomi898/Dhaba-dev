import { request } from './client';

export const menuApi = {
  list: (opts) => request('/api/menu', opts),
  get: (id, opts) => request(`/api/menu/${encodeURIComponent(id)}`, opts),
};

export const ordersApi = {
  create: (order) => request('/api/orders', { method: 'POST', body: order }),
  get: (id, opts) => request(`/api/orders/${encodeURIComponent(id)}`, opts),
  list: ({ status, limit } = {}, opts) => {
    const params = new URLSearchParams();
    if (status?.length) params.set('status', status.join(','));
    if (limit) params.set('limit', String(limit));
    const qs = params.toString();
    return request(`/api/orders${qs ? `?${qs}` : ''}`, opts);
  },
  setStatus: (id, status) => request(`/api/orders/${encodeURIComponent(id)}/status`, { method: 'PATCH', body: { status } }),
};

export const statsApi = {
  summary: (opts) => request('/api/stats', opts),
  topDishes: (limit, opts) => request(`/api/stats/top-dishes?limit=${limit}`, opts),
};

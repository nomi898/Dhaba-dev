import { vi } from 'vitest';

export const MENU = [
  {
    id: 1, name: 'Karahi & Curry', slug: 'karahi', items: [
      { id: 4, name: 'Chicken Karahi', description: 'Wok curry', priceCents: 145000, isAvailable: true, isVegetarian: false, spiceLevel: 2 },
      { id: 8, name: 'Nihari', description: 'Beef stew', priceCents: 120000, isAvailable: false, isVegetarian: false, spiceLevel: 3 },
    ],
  },
  {
    id: 2, name: 'Breads', slug: 'breads', items: [
      { id: 15, name: 'Garlic Naan', description: 'Buttery', priceCents: 12000, isAvailable: true, isVegetarian: true, spiceLevel: 0 },
    ],
  },
];

const json = (status, body) => ({ ok: status < 400, status, text: async () => JSON.stringify(body) });

/** Route-based fetch mock: routes = { 'GET /api/menu': (body) => [status, payload] } */
export function mockFetch(routes) {
  const calls = [];
  const fn = vi.fn(async (url, opts = {}) => {
    const method = opts.method || 'GET';
    const body = opts.body ? JSON.parse(opts.body) : undefined;
    calls.push({ method, url, body });
    const key = Object.keys(routes).find((k) => {
      const [m, path] = k.split(' ');
      return m === method && (path.endsWith('*') ? url.startsWith(path.slice(0, -1)) : url === path);
    });
    if (!key) return json(404, { error: { code: 'not_found', message: `no mock for ${method} ${url}` } });
    const [status, payload] = routes[key](body, url);
    return json(status, payload);
  });
  vi.stubGlobal('fetch', fn);
  return calls;
}

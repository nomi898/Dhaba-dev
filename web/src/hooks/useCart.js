import { useCallback, useReducer } from 'react';
import { cartReducer, emptyCart } from '../lib/cart';

const KEY = 'dhaba.cart.v1';

function load() {
  try {
    const parsed = JSON.parse(localStorage.getItem(KEY));
    return Array.isArray(parsed?.lines) ? parsed : emptyCart;
  } catch {
    return emptyCart;
  }
}

function save(cart) {
  try {
    localStorage.setItem(KEY, JSON.stringify(cart));
  } catch {
    /* storage full or disabled: cart still works in memory */
  }
}

function persistingReducer(state, action) {
  const next = cartReducer(state, action);
  if (next !== state) save(next);
  return next;
}

export function useCart() {
  const [cart, rawDispatch] = useReducer(persistingReducer, undefined, load);
  // React runs reducers lazily on the next render. Clearing is usually followed by navigation,
  // which unmounts the page before that render, so persist the empty cart synchronously.
  const dispatch = useCallback((action) => {
    if (action.type === 'clear') save(emptyCart);
    rawDispatch(action);
  }, []);
  return [cart, dispatch];
}

const ORDERS_KEY = 'dhaba.myOrders.v1';

export function rememberOrder(id) {
  try {
    const ids = JSON.parse(localStorage.getItem(ORDERS_KEY)) || [];
    localStorage.setItem(ORDERS_KEY, JSON.stringify([id, ...ids.filter((x) => x !== id)].slice(0, 10)));
  } catch {
    /* ignore */
  }
}

export function myOrders() {
  try {
    const ids = JSON.parse(localStorage.getItem(ORDERS_KEY));
    return Array.isArray(ids) ? ids : [];
  } catch {
    return [];
  }
}

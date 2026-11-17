import { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { menuApi, ordersApi } from '../api/services';
import CartPanel from '../components/CartPanel';
import MenuItemCard from '../components/MenuItemCard';
import { ErrorBanner, Spinner } from '../components/common';
import { rememberOrder, useCart } from '../hooks/useCart';
import { usePolling } from '../hooks/usePolling';
import { quantityOf, toOrderPayload } from '../lib/cart';

export default function OrderPage() {
  const { data: menu, error, loading, refresh } = usePolling((signal) => menuApi.list({ signal }), 60000);
  const [cart, dispatch] = useCart();
  const [active, setActive] = useState('all');
  const [submitting, setSubmitting] = useState(false);
  const [serverError, setServerError] = useState(null);
  const navigate = useNavigate();

  const allItems = useMemo(() => (menu || []).flatMap((c) => c.items), [menu]);
  useEffect(() => {
    if (menu) dispatch({ type: 'sync', items: allItems });
  }, [menu, allItems, dispatch]);

  const categories = (menu || []).filter((c) => c.items.length > 0);
  const shown = active === 'all' ? categories : categories.filter((c) => c.slug === active);

  const checkout = async (form) => {
    setSubmitting(true);
    setServerError(null);
    try {
      const order = await ordersApi.create(toOrderPayload(cart, form));
      rememberOrder(order.id);
      dispatch({ type: 'clear' });
      navigate(`/orders/${order.id}`, { state: { justPlaced: true } });
    } catch (err) {
      setServerError(err.fields ? Object.values(err.fields).join(' ') : err.message);
      refresh();
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="order-layout">
      <section className="menu" aria-label="Menu">
        <header className="page-head">
          <h1>Menu</h1>
          <p className="muted">Fresh from the tandoor. Prices include tax.</p>
        </header>
        <ErrorBanner error={error} onRetry={refresh} />
        {loading && !menu && <Spinner label="Loading menu…" />}
        {categories.length > 0 && (
          <nav className="chips" aria-label="Categories">
            <button type="button" className={`chip ${active === 'all' ? 'chip-on' : ''}`} onClick={() => setActive('all')}>All</button>
            {categories.map((c) => (
              <button key={c.id} type="button" className={`chip ${active === c.slug ? 'chip-on' : ''}`} onClick={() => setActive(c.slug)}>
                {c.name}
              </button>
            ))}
          </nav>
        )}
        {shown.map((c) => (
          <section key={c.id} className="category" aria-labelledby={`cat-${c.id}`}>
            <h2 id={`cat-${c.id}`}>{c.name}</h2>
            <div className="menu-grid">
              {c.items.map((item) => (
                <MenuItemCard
                  key={item.id}
                  item={item}
                  quantity={quantityOf(cart, item.id)}
                  onAdd={() => dispatch({ type: 'add', item })}
                  onDecrement={() => dispatch({ type: 'decrement', id: item.id })}
                />
              ))}
            </div>
          </section>
        ))}
      </section>
      <CartPanel cart={cart} dispatch={dispatch} onCheckout={checkout} submitting={submitting} serverError={serverError} />
    </div>
  );
}

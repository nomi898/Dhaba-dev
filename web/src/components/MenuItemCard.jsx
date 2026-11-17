import { formatPrice } from '../lib/format';

const SPICE = ['', '🌶', '🌶🌶', '🌶🌶🌶'];

export default function MenuItemCard({ item, quantity, onAdd, onDecrement }) {
  return (
    <article className={`menu-card ${item.isAvailable ? '' : 'menu-card-off'}`}>
      <div className="menu-card-body">
        <h3>
          {item.name}
          {item.isVegetarian && <span className="tag tag-veg" title="Vegetarian">Veg</span>}
          {item.spiceLevel > 0 && <span className="spice" aria-label={`Spice level ${item.spiceLevel} of 3`}>{SPICE[item.spiceLevel]}</span>}
        </h3>
        <p>{item.description}</p>
      </div>
      <div className="menu-card-foot">
        <span className="price">{formatPrice(item.priceCents)}</span>
        {!item.isAvailable ? (
          <span className="tag tag-off">Unavailable</span>
        ) : quantity > 0 ? (
          <div className="stepper" aria-label={`${item.name} quantity`}>
            <button type="button" onClick={onDecrement} aria-label={`Remove one ${item.name}`}>−</button>
            <span aria-live="polite">{quantity}</span>
            <button type="button" onClick={onAdd} aria-label={`Add one more ${item.name}`}>+</button>
          </div>
        ) : (
          <button type="button" className="btn btn-primary btn-sm" onClick={onAdd} aria-label={`Add ${item.name}`}>
            Add
          </button>
        )}
      </div>
    </article>
  );
}

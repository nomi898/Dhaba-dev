import { NavLink, Route, Routes } from 'react-router-dom';
import KitchenPage from './pages/KitchenPage';
import OrderPage from './pages/OrderPage';
import StatsPage from './pages/StatsPage';
import TrackPage from './pages/TrackPage';

const link = ({ isActive }) => `nav-link ${isActive ? 'nav-link-on' : ''}`;

export default function App() {
  return (
    <div className="app">
      <header className="topbar">
        <NavLink to="/" className="brand">
          <img src="/favicon.svg" alt="" width="28" height="28" />
          Dhaba Orders
        </NavLink>
        <nav aria-label="Main">
          <NavLink to="/" end className={link}>Order</NavLink>
          <NavLink to="/orders" className={link}>Track</NavLink>
          <NavLink to="/kitchen" className={link}>Kitchen</NavLink>
          <NavLink to="/stats" className={link}>Stats</NavLink>
        </nav>
      </header>
      <main className="container">
        <Routes>
          <Route path="/" element={<OrderPage />} />
          <Route path="/orders" element={<TrackPage />} />
          <Route path="/orders/:id" element={<TrackPage />} />
          <Route path="/kitchen" element={<KitchenPage />} />
          <Route path="/stats" element={<StatsPage />} />
          <Route path="*" element={<div className="narrow"><h1>Page not found</h1></div>} />
        </Routes>
      </main>
    </div>
  );
}

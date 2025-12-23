import { NavLink, Outlet } from 'react-router-dom';
import './Layout.css';

export default function Layout() {
  return (
    <div className="layout">
      <header className="header">
        <h1 className="logo">📺 TV Tracker</h1>
        <nav className="nav">
          <NavLink to="/" className={({ isActive }) => isActive ? 'nav-link active' : 'nav-link'}>
            看板
          </NavLink>
          <NavLink to="/search" className={({ isActive }) => isActive ? 'nav-link active' : 'nav-link'}>
            搜索
          </NavLink>
          <NavLink to="/library" className={({ isActive }) => isActive ? 'nav-link active' : 'nav-link'}>
            片库
          </NavLink>
        </nav>
      </header>
      <main className="main">
        <Outlet />
      </main>
    </div>
  );
}

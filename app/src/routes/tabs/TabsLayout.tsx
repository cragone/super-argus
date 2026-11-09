import { NavLink, Outlet } from 'react-router-dom';
import type { CSSProperties } from 'react';

import { useTheme } from '@/theme';

const TABS = [
  { to: '/services', label: 'Services' },
  { to: '/tail', label: 'Live Tail' },
  { to: '/dashboards', label: 'Dashboards' },
  { to: '/alerts', label: 'Alerts' },
  { to: '/settings', label: 'Settings' },
];

export function TabsLayout() {
  const t = useTheme();

  const nav: CSSProperties = {
    display: 'flex',
    gap: 4,
    padding: '10px 16px',
    backgroundColor: t.surface,
    borderBottom: `1px solid ${t.border}`,
    overflowX: 'auto',
  };

  const linkStyle = (active: boolean): CSSProperties => ({
    padding: '8px 14px',
    borderRadius: 8,
    fontSize: 15,
    fontWeight: 600,
    textDecoration: 'none',
    whiteSpace: 'nowrap',
    color: active ? t.accentText : t.textDim,
    backgroundColor: active ? t.accent : 'transparent',
  });

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <nav style={nav}>
        {TABS.map((tab) => (
          <NavLink key={tab.to} to={tab.to} style={({ isActive }) => linkStyle(isActive)}>
            {tab.label}
          </NavLink>
        ))}
      </nav>
      <div style={{ flex: 1, overflow: 'auto', backgroundColor: t.bg }}>
        <Outlet />
      </div>
    </div>
  );
}

import { Navigate, Route, Routes } from 'react-router-dom';

import { ProtectedRoute } from '@/auth/ProtectedRoute';
import { PublicRoute } from '@/auth/PublicRoute';
import { Login } from '@/routes/Login';
import { NotFound } from '@/routes/NotFound';
import { Alerts } from '@/routes/tabs/Alerts';
import { Settings } from '@/routes/tabs/Settings';
import { TabsLayout } from '@/routes/tabs/TabsLayout';
import { Tail } from '@/routes/tabs/Tail';
import { DashboardDetail } from '@/routes/tabs/dashboards/DashboardDetail';
import { DashboardsIndex } from '@/routes/tabs/dashboards/DashboardsIndex';
import { DashboardsLayout } from '@/routes/tabs/dashboards/DashboardsLayout';
import { ServiceDetail } from '@/routes/tabs/services/ServiceDetail';
import { ServicesIndex } from '@/routes/tabs/services/ServicesIndex';
import { ServicesLayout } from '@/routes/tabs/services/ServicesLayout';

export function AppRoutes() {
  return (
    <Routes>
      <Route element={<PublicRoute />}>
        <Route path="/login" element={<Login />} />
      </Route>

      <Route element={<ProtectedRoute />}>
        <Route path="/" element={<Navigate to="/services" replace />} />
        <Route element={<TabsLayout />}>
          <Route path="/services" element={<ServicesLayout />}>
            <Route index element={<ServicesIndex />} />
            <Route path=":name" element={<ServiceDetail />} />
          </Route>
          <Route path="/tail" element={<Tail />} />
          <Route path="/dashboards" element={<DashboardsLayout />}>
            <Route index element={<DashboardsIndex />} />
            <Route path=":id" element={<DashboardDetail />} />
          </Route>
          <Route path="/alerts" element={<Alerts />} />
          <Route path="/settings" element={<Settings />} />
        </Route>
      </Route>

      <Route path="*" element={<NotFound />} />
    </Routes>
  );
}

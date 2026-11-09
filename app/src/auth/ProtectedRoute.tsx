import { Navigate, Outlet } from 'react-router-dom';

import { useSession } from './session';

/** Wraps routes that require a signed-in session; `status === 'loading'` never reaches here (App.tsx short-circuits it). */
export function ProtectedRoute() {
  const status = useSession((s) => s.status);
  if (status !== 'signedIn') return <Navigate to="/login" replace />;
  return <Outlet />;
}

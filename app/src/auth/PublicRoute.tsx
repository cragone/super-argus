import { Navigate, Outlet } from 'react-router-dom';

import { useSession } from './session';

/** Wraps routes (login) that redirect away once already signed in. */
export function PublicRoute() {
  const status = useSession((s) => s.status);
  if (status === 'signedIn') return <Navigate to="/" replace />;
  return <Outlet />;
}

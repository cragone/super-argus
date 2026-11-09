import { Navigate } from 'react-router-dom';

import { useSession } from '@/auth/session';

export function NotFound() {
  const status = useSession((s) => s.status);
  return <Navigate to={status === 'signedIn' ? '/services' : '/login'} replace />;
}

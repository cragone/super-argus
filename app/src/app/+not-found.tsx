import { Redirect } from 'expo-router';

import { useSession } from '@/auth/session';

export default function NotFoundScreen() {
  const status = useSession((s) => s.status);
  return <Redirect href={status === 'signedIn' ? '/services' : '/login'} />;
}

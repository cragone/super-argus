import { useSession } from '@/auth/session';
import { ChannelsCard } from '@/components/ChannelsCard';
import { Button, Card } from '@/components/ui';
import { useIsAdmin } from '@/hooks/useAlerts';
import { useDocumentTitle } from '@/hooks/useDocumentTitle';
import { useTheme } from '@/theme';

export function Settings() {
  useDocumentTitle('Settings');
  const t = useTheme();
  const user = useSession((s) => s.user);
  const origin = useSession((s) => s.origin);
  const signOut = useSession((s) => s.signOut);
  const isAdmin = useIsAdmin();

  return (
    <div style={{ padding: 16, display: 'flex', flexDirection: 'column', gap: 16 }}>
      <Card style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
        <span style={{ color: t.textDim }}>Signed in as</span>
        <span style={{ color: t.text, fontSize: 17, fontWeight: 600 }}>{user?.email}</span>
        <span style={{ color: t.textDim }}>Role: {user?.role}</span>
        <span style={{ color: t.textDim }}>Server: {origin || 'this site'}</span>
      </Card>
      {isAdmin && <ChannelsCard />}
      <Button title="Sign out" variant="ghost" onPress={() => void signOut()} />
    </div>
  );
}

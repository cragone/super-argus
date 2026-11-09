import { useState, type CSSProperties } from 'react';

import { errorMessage } from '@/api/client';
import { useSession } from '@/auth/session';
import { Button, ErrorText } from '@/components/ui';
import { AsciiOwl } from '@/components/AsciiOwl';
import { useDocumentTitle } from '@/hooks/useDocumentTitle';
import { useTheme } from '@/theme';

export function Login() {
  useDocumentTitle('Sign in');
  const t = useTheme();
  const signIn = useSession((s) => s.signIn);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const canSubmit = !!email.trim() && !!password;

  const submit = async (e?: React.FormEvent) => {
    e?.preventDefault();
    if (!canSubmit || busy) return;
    setBusy(true);
    setError(null);
    try {
      await signIn(email.trim(), password);
    } catch (err) {
      setError(errorMessage(err, 'Sign in failed'));
    } finally {
      setBusy(false);
    }
  };

  const field: CSSProperties = {
    color: t.text,
    backgroundColor: t.surface,
    borderColor: t.border,
    borderWidth: 1,
    borderStyle: 'solid',
    borderRadius: 8,
    padding: '10px 12px',
    fontSize: 16,
    width: '100%',
  };

  const card: CSSProperties = {
    width: '100%',
    maxWidth: 400,
    margin: '0 auto',
    display: 'flex',
    flexDirection: 'column',
    gap: 14,
    padding: 24,
    borderRadius: 16,
    borderWidth: 1,
    borderStyle: 'solid',
    borderColor: t.border,
    // Translucent so the ASCII owl shows around and faintly through it.
    backgroundColor: `${t.bg}e6`,
  };

  return (
    <div style={{ position: 'relative', minHeight: '100%', backgroundColor: t.bg }}>
      <AsciiOwl />
      <div
        style={{
          position: 'relative',
          minHeight: '100%',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          padding: 24,
        }}>
        <form style={card} onSubmit={submit}>
          <h1 style={{ fontSize: 30, fontWeight: 700, color: t.text, margin: 0 }}>Argus</h1>
          <p style={{ color: t.textDim, margin: 0 }}>Sign in to your log server.</p>

          <input
            aria-label="Username"
            placeholder="Username"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            autoCapitalize="none"
            autoCorrect="off"
            autoComplete="username"
            style={field}
          />
          <input
            aria-label="Password"
            placeholder="Password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete="current-password"
            style={field}
          />
          {error && <ErrorText>{error}</ErrorText>}
          <Button title="Sign in" onPress={() => void submit()} disabled={!canSubmit} loading={busy} />
        </form>
      </div>
    </div>
  );
}

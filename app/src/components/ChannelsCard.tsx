import { useState, type CSSProperties } from 'react';

import { errorMessage, type Channel, type ChannelInput } from '@/api/client';
import { useAlertMutations, useChannels } from '@/hooks/useAlerts';
import { useTheme } from '@/theme';

import { Button, Card, Chip, ErrorText } from './ui';

type ChannelType = ChannelInput['type'];

const CHANNEL_TYPES: { id: ChannelType; label: string }[] = [
  { id: 'webhook', label: 'Webhook' },
  { id: 'email', label: 'Email' },
  { id: 'sms', label: 'SMS' },
];

/** Comma-separated recipients -> a trimmed, non-empty list. */
function toList(s: string): string[] {
  return s.split(',').map((x) => x.trim()).filter(Boolean);
}

/** One line describing a channel, by type, for the list view. */
function summarize(c: Channel): string {
  switch (c.type) {
    case 'email':
      return `email · ${c.smtp_host ?? ''} → ${c.from ?? ''}`;
    case 'sms':
      return `sms · ${c.account_sid ?? ''} → ${c.from ?? ''}`;
    default:
      return `webhook · ${c.url_host ?? ''}`;
  }
}

/** Admin-only management of notification channels (webhook, email, SMS). */
export function ChannelsCard() {
  const t = useTheme();
  const channels = useChannels();
  const { createChannel, deleteChannel, testChannel } = useAlertMutations();
  const [type, setType] = useState<ChannelType>('webhook');
  const [name, setName] = useState('');
  const [url, setUrl] = useState('');
  const [smtpHost, setSmtpHost] = useState('');
  const [smtpPort, setSmtpPort] = useState('');
  const [username, setUsername] = useState('');
  const [accountSid, setAccountSid] = useState('');
  const [from, setFrom] = useState('');
  const [to, setTo] = useState('');
  const [secret, setSecret] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [results, setResults] = useState<Record<string, string>>({});

  const reset = () => {
    setName(''); setUrl(''); setSmtpHost(''); setSmtpPort(''); setUsername('');
    setAccountSid(''); setFrom(''); setTo(''); setSecret('');
  };

  const add = () => {
    setError(null);
    if (!name.trim()) return setError('Name is required');
    const withSecret = (body: ChannelInput) => (secret ? { ...body, secret } : body);

    let body: ChannelInput;
    if (type === 'webhook') {
      if (!url.trim()) return setError('URL is required');
      body = withSecret({ name: name.trim(), type: 'webhook', url: url.trim() });
    } else if (type === 'email') {
      const port = Number(smtpPort);
      if (!smtpHost.trim()) return setError('SMTP host is required');
      if (!smtpPort.trim() || !Number.isInteger(port) || port < 1 || port > 65535) return setError('SMTP port must be between 1 and 65535');
      if (!from.trim()) return setError('From address is required');
      const recipients = toList(to);
      if (recipients.length === 0) return setError('At least one recipient is required');
      body = withSecret({
        name: name.trim(), type: 'email', smtp_host: smtpHost.trim(), smtp_port: port,
        ...(username.trim() ? { username: username.trim() } : {}), from: from.trim(), to: recipients,
      });
    } else {
      if (!accountSid.trim()) return setError('Account SID is required');
      if (!from.trim()) return setError('From number is required');
      const recipients = toList(to);
      if (recipients.length === 0) return setError('At least one recipient is required');
      body = withSecret({ name: name.trim(), type: 'sms', account_sid: accountSid.trim(), from: from.trim(), to: recipients });
    }

    createChannel.mutate(body, {
      onSuccess: reset,
      onError: (e) => setError(errorMessage(e, 'Could not add the channel')),
    });
  };
  const test = (id: string) =>
    testChannel.mutate(id, {
      onSuccess: (r) => setResults((m) => ({ ...m, [id]: r.ok ? 'Delivered' : `Failed: ${r.error ?? 'unknown error'}` })),
      onError: (e) => setResults((m) => ({ ...m, [id]: errorMessage(e, 'Test failed') })),
    });

  const field: CSSProperties = {
    color: t.text, backgroundColor: t.surface, borderColor: t.border, borderWidth: 1, borderStyle: 'solid',
    borderRadius: 8, padding: '8px 10px', fontSize: 14, width: '100%',
  };
  return (
    <Card style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
      <span style={{ color: t.text, fontSize: 16, fontWeight: 700 }}>Notification channels</span>
      {channels.data?.length === 0 && <span style={{ color: t.textDim }}>No channels yet.</span>}
      {channels.data?.map((c) => (
        <div key={c.id} style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
          <span style={{ color: t.text, fontWeight: 600 }}>{c.name}</span>
          <span style={{ color: t.textDim, fontSize: 12 }}>
            {summarize(c)}
            {c.has_secret ? ' · signed' : ''}
          </span>
          <div style={{ display: 'flex', gap: 8 }}>
            <Chip label="Send test" accessibilityLabel={`Test ${c.name}`} onPress={() => test(c.id)} />
            <Chip
              label="Delete"
              accessibilityLabel={`Delete channel ${c.name}`}
              onPress={() => deleteChannel.mutate(c.id, { onError: (e) => setError(errorMessage(e, 'Could not delete the channel')) })}
            />
          </div>
          {results[c.id] && <span style={{ color: t.textDim, fontSize: 12 }}>{results[c.id]}</span>}
        </div>
      ))}

      <div style={{ display: 'flex', gap: 6 }}>
        {CHANNEL_TYPES.map((ct) => (
          <Chip
            key={ct.id}
            label={ct.label}
            selected={type === ct.id}
            accessibilityLabel={`Channel type ${ct.label}`}
            onPress={() => setType(ct.id)}
          />
        ))}
      </div>
      <input aria-label="Channel name" placeholder="Name" value={name} onChange={(e) => setName(e.target.value)} style={field} />

      {type === 'webhook' && (
        <input aria-label="Webhook URL" placeholder="https://example.com/hook" value={url} onChange={(e) => setUrl(e.target.value)} autoCapitalize="none" autoCorrect="off" style={field} />
      )}

      {type === 'email' && (
        <>
          <input aria-label="SMTP host" placeholder="smtp.example.com" value={smtpHost} onChange={(e) => setSmtpHost(e.target.value)} autoCapitalize="none" autoCorrect="off" style={field} />
          <input aria-label="SMTP port" placeholder="587" value={smtpPort} onChange={(e) => setSmtpPort(e.target.value)} inputMode="numeric" style={field} />
          <input aria-label="SMTP username" placeholder="Username (optional)" value={username} onChange={(e) => setUsername(e.target.value)} autoCapitalize="none" autoCorrect="off" style={field} />
          <input aria-label="From address" placeholder="alerts@example.com" value={from} onChange={(e) => setFrom(e.target.value)} autoCapitalize="none" autoCorrect="off" style={field} />
          <input aria-label="Recipients" placeholder="oncall@example.com, lead@example.com" value={to} onChange={(e) => setTo(e.target.value)} autoCapitalize="none" autoCorrect="off" style={field} />
        </>
      )}

      {type === 'sms' && (
        <>
          <input aria-label="Twilio account SID" placeholder="ACxxxxxxxxxxxxxxxx" value={accountSid} onChange={(e) => setAccountSid(e.target.value)} autoCapitalize="none" autoCorrect="off" style={field} />
          <input aria-label="From number" placeholder="+15550000000" value={from} onChange={(e) => setFrom(e.target.value)} autoCapitalize="none" autoCorrect="off" style={field} />
          <input aria-label="Recipients" placeholder="+15551111111, +15552222222" value={to} onChange={(e) => setTo(e.target.value)} autoCapitalize="none" autoCorrect="off" style={field} />
        </>
      )}

      <input
        aria-label="Secret"
        placeholder={type === 'webhook' ? 'Signing secret (optional)' : type === 'email' ? 'SMTP password (optional)' : 'Auth token (optional)'}
        value={secret}
        onChange={(e) => setSecret(e.target.value)}
        autoCapitalize="none"
        type="password"
        style={field}
      />
      {error && <ErrorText>{error}</ErrorText>}
      <Button title="Add channel" onPress={add} loading={createChannel.isPending} />
    </Card>
  );
}

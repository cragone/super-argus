import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import type { ReactElement } from 'react';
import { vi, type Mock } from 'vitest';

import { api, type Channel } from '@/api/client';
import { ChannelsCard } from '@/components/ChannelsCard';
import { useSession } from '@/auth/session';

vi.mock('@/auth/session', () => ({ useSession: vi.fn() }));

const withClient = (ui: ReactElement) => (
  <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>{ui}</QueryClientProvider>
);

const channels: Channel[] = [
  { id: 'w1', name: 'ops', type: 'webhook', url_host: 'h.io', has_secret: false, created_at: '', updated_at: '' },
  { id: 'e1', name: 'oncall', type: 'email', smtp_host: 'smtp.example.com', from: 'alerts@example.com', has_secret: true, created_at: '', updated_at: '' },
  { id: 's1', name: 'sms-oncall', type: 'sms', account_sid: 'ACxxxx', from: '+15550000000', has_secret: true, created_at: '', updated_at: '' },
];

beforeEach(() => {
  (useSession as unknown as Mock).mockImplementation((sel: (s: unknown) => unknown) =>
    sel({ status: 'signedIn', user: { role: 'admin' } }),
  );
  vi.spyOn(api, 'GET').mockResolvedValue({ data: channels, error: undefined, response: new Response() } as never);
});

afterEach(() => vi.restoreAllMocks());

describe('ChannelsCard', () => {
  it('renders each channel with a type-specific summary line', async () => {
    render(withClient(<ChannelsCard />));
    expect(await screen.findByText(/webhook · h\.io/)).toBeTruthy();
    expect(await screen.findByText(/email · smtp\.example\.com → alerts@example\.com/)).toBeTruthy();
    expect(await screen.findByText(/sms · ACxxxx → \+15550000000/)).toBeTruthy();
  });

  it('shows only the fields for the selected channel type', async () => {
    render(withClient(<ChannelsCard />));
    await screen.findByText(/webhook · h\.io/);
    expect(screen.getByLabelText('Webhook URL')).toBeTruthy();
    expect(screen.queryByLabelText('SMTP host')).toBeNull();

    fireEvent.click(screen.getByLabelText('Channel type Email'));
    expect(screen.getByLabelText('SMTP host')).toBeTruthy();
    expect(screen.getByLabelText('SMTP port')).toBeTruthy();
    expect(screen.queryByLabelText('Webhook URL')).toBeNull();
    expect(screen.queryByLabelText('Twilio account SID')).toBeNull();

    fireEvent.click(screen.getByLabelText('Channel type SMS'));
    expect(screen.getByLabelText('Twilio account SID')).toBeTruthy();
    expect(screen.queryByLabelText('SMTP host')).toBeNull();
  });

  it('submits an email channel with recipients split into a list', async () => {
    const post = vi.spyOn(api, 'POST').mockResolvedValue({ data: channels[1], error: undefined, response: new Response() } as never);
    render(withClient(<ChannelsCard />));
    await screen.findByText(/webhook · h\.io/);

    fireEvent.click(screen.getByLabelText('Channel type Email'));
    fireEvent.change(screen.getByLabelText('Channel name'), { target: { value: 'oncall' } });
    fireEvent.change(screen.getByLabelText('SMTP host'), { target: { value: 'smtp.example.com' } });
    fireEvent.change(screen.getByLabelText('SMTP port'), { target: { value: '587' } });
    fireEvent.change(screen.getByLabelText('From address'), { target: { value: 'alerts@example.com' } });
    fireEvent.change(screen.getByLabelText('Recipients'), { target: { value: 'oncall@example.com, lead@example.com' } });
    fireEvent.click(screen.getByLabelText('Add channel'));

    await waitFor(() =>
      expect(post).toHaveBeenCalledWith(
        '/channels',
        expect.objectContaining({
          body: {
            name: 'oncall',
            type: 'email',
            smtp_host: 'smtp.example.com',
            smtp_port: 587,
            from: 'alerts@example.com',
            to: ['oncall@example.com', 'lead@example.com'],
          },
        }),
      ),
    );
  });

  it('submits an sms channel with recipients split into a list', async () => {
    const post = vi.spyOn(api, 'POST').mockResolvedValue({ data: channels[2], error: undefined, response: new Response() } as never);
    render(withClient(<ChannelsCard />));
    await screen.findByText(/webhook · h\.io/);

    fireEvent.click(screen.getByLabelText('Channel type SMS'));
    fireEvent.change(screen.getByLabelText('Channel name'), { target: { value: 'sms-oncall' } });
    fireEvent.change(screen.getByLabelText('Twilio account SID'), { target: { value: 'ACxxxx' } });
    fireEvent.change(screen.getByLabelText('From number'), { target: { value: '+15550000000' } });
    fireEvent.change(screen.getByLabelText('Recipients'), { target: { value: '+15551111111, +15552222222' } });
    fireEvent.click(screen.getByLabelText('Add channel'));

    await waitFor(() =>
      expect(post).toHaveBeenCalledWith(
        '/channels',
        expect.objectContaining({
          body: {
            name: 'sms-oncall',
            type: 'sms',
            account_sid: 'ACxxxx',
            from: '+15550000000',
            to: ['+15551111111', '+15552222222'],
          },
        }),
      ),
    );
  });

  it('rejects an email channel missing a recipient before calling the server', async () => {
    const post = vi.spyOn(api, 'POST');
    render(withClient(<ChannelsCard />));
    await screen.findByText(/webhook · h\.io/);

    fireEvent.click(screen.getByLabelText('Channel type Email'));
    fireEvent.change(screen.getByLabelText('Channel name'), { target: { value: 'oncall' } });
    fireEvent.change(screen.getByLabelText('SMTP host'), { target: { value: 'smtp.example.com' } });
    fireEvent.change(screen.getByLabelText('SMTP port'), { target: { value: '587' } });
    fireEvent.change(screen.getByLabelText('From address'), { target: { value: 'alerts@example.com' } });
    fireEvent.click(screen.getByLabelText('Add channel'));

    expect(await screen.findByRole('alert')).toBeTruthy();
    expect(post).not.toHaveBeenCalled();
  });
});

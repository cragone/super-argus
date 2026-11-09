import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen, waitFor } from '@testing-library/react-native';
import type { ReactElement } from 'react';

import { api, type Channel } from '@/api/client';
import { ChannelsCard } from '@/components/ChannelsCard';
import { useSession } from '@/auth/session';

jest.mock('@/auth/session', () => ({ useSession: jest.fn() }));

const withClient = (ui: ReactElement) => (
  <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>{ui}</QueryClientProvider>
);

const channels: Channel[] = [
  { id: 'w1', name: 'ops', type: 'webhook', url_host: 'h.io', has_secret: false, created_at: '', updated_at: '' },
  { id: 'e1', name: 'oncall', type: 'email', smtp_host: 'smtp.example.com', from: 'alerts@example.com', has_secret: true, created_at: '', updated_at: '' },
  { id: 's1', name: 'sms-oncall', type: 'sms', account_sid: 'ACxxxx', from: '+15550000000', has_secret: true, created_at: '', updated_at: '' },
];

beforeEach(() => {
  (useSession as unknown as jest.Mock).mockImplementation((sel: (s: unknown) => unknown) =>
    sel({ status: 'signedIn', user: { role: 'admin' } }),
  );
  jest.spyOn(api, 'GET').mockResolvedValue({ data: channels, error: undefined, response: new Response() } as never);
});

afterEach(() => jest.restoreAllMocks());

describe('ChannelsCard', () => {
  it('renders each channel with a type-specific summary line', async () => {
    await render(withClient(<ChannelsCard />));
    expect(await screen.findByText(/webhook · h\.io/)).toBeTruthy();
    expect(await screen.findByText(/email · smtp\.example\.com → alerts@example\.com/)).toBeTruthy();
    expect(await screen.findByText(/sms · ACxxxx → \+15550000000/)).toBeTruthy();
  });

  it('shows only the fields for the selected channel type', async () => {
    await render(withClient(<ChannelsCard />));
    await screen.findByText(/webhook · h\.io/);
    expect(screen.getByLabelText('Webhook URL')).toBeTruthy();
    expect(screen.queryByLabelText('SMTP host')).toBeNull();

    await fireEvent.press(screen.getByLabelText('Channel type Email'));
    expect(screen.getByLabelText('SMTP host')).toBeTruthy();
    expect(screen.getByLabelText('SMTP port')).toBeTruthy();
    expect(screen.queryByLabelText('Webhook URL')).toBeNull();
    expect(screen.queryByLabelText('Twilio account SID')).toBeNull();

    await fireEvent.press(screen.getByLabelText('Channel type SMS'));
    expect(screen.getByLabelText('Twilio account SID')).toBeTruthy();
    expect(screen.queryByLabelText('SMTP host')).toBeNull();
  });

  it('submits an email channel with recipients split into a list', async () => {
    const post = jest.spyOn(api, 'POST').mockResolvedValue({ data: channels[1], error: undefined, response: new Response() } as never);
    await render(withClient(<ChannelsCard />));
    await screen.findByText(/webhook · h\.io/);

    await fireEvent.press(screen.getByLabelText('Channel type Email'));
    await fireEvent.changeText(screen.getByLabelText('Channel name'), 'oncall');
    await fireEvent.changeText(screen.getByLabelText('SMTP host'), 'smtp.example.com');
    await fireEvent.changeText(screen.getByLabelText('SMTP port'), '587');
    await fireEvent.changeText(screen.getByLabelText('From address'), 'alerts@example.com');
    await fireEvent.changeText(screen.getByLabelText('Recipients'), 'oncall@example.com, lead@example.com');
    await fireEvent.press(screen.getByLabelText('Add channel'));

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
    const post = jest.spyOn(api, 'POST').mockResolvedValue({ data: channels[2], error: undefined, response: new Response() } as never);
    await render(withClient(<ChannelsCard />));
    await screen.findByText(/webhook · h\.io/);

    await fireEvent.press(screen.getByLabelText('Channel type SMS'));
    await fireEvent.changeText(screen.getByLabelText('Channel name'), 'sms-oncall');
    await fireEvent.changeText(screen.getByLabelText('Twilio account SID'), 'ACxxxx');
    await fireEvent.changeText(screen.getByLabelText('From number'), '+15550000000');
    await fireEvent.changeText(screen.getByLabelText('Recipients'), '+15551111111, +15552222222');
    await fireEvent.press(screen.getByLabelText('Add channel'));

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
    const post = jest.spyOn(api, 'POST');
    await render(withClient(<ChannelsCard />));
    await screen.findByText(/webhook · h\.io/);

    await fireEvent.press(screen.getByLabelText('Channel type Email'));
    await fireEvent.changeText(screen.getByLabelText('Channel name'), 'oncall');
    await fireEvent.changeText(screen.getByLabelText('SMTP host'), 'smtp.example.com');
    await fireEvent.changeText(screen.getByLabelText('SMTP port'), '587');
    await fireEvent.changeText(screen.getByLabelText('From address'), 'alerts@example.com');
    await fireEvent.press(screen.getByLabelText('Add channel'));

    expect(await screen.findByRole('alert')).toBeTruthy();
    expect(post).not.toHaveBeenCalled();
  });
});

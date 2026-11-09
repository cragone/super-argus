import { create } from 'zustand';

import { api, configureClient, ApiError, unwrap, type Me } from '@/api/client';

export type SessionStatus = 'loading' | 'signedOut' | 'signedIn';

interface SessionState {
  status: SessionStatus;
  /** Server origin. Empty means same-origin; only set in dev via VITE_ARGUS_URL. */
  origin: string;
  accessToken: string | null;
  user: Me | null;
  bootstrap(): Promise<void>;
  signIn(email: string, password: string): Promise<void>;
  signOut(): Promise<void>;
}

/** Normalises user input like "10.0.0.5:8080" into "http://10.0.0.5:8080". */
export function normalizeServerUrl(input: string): string {
  let s = input.trim().replace(/\/+$/, '');
  if (!s) return '';
  if (!/^[a-z][a-z0-9+.-]*:\/\//i.test(s)) s = `http://${s}`;
  return s;
}

const devOrigin = (import.meta.env.VITE_ARGUS_URL ?? '').replace(/\/+$/, '');

let refreshInFlight: Promise<string | null> | null = null;

export const useSession = create<SessionState>((set) => ({
  status: 'loading',
  origin: devOrigin,
  accessToken: null,
  user: null,

  async bootstrap() {
    const token = await refreshAccessToken();
    if (!token) {
      set({ status: 'signedOut' });
      return;
    }
    try {
      const user = unwrap(await api.GET('/me'));
      set({ user, status: 'signedIn' });
    } catch {
      set({ status: 'signedOut', accessToken: null });
    }
  },

  async signIn(email, password) {
    let res;
    try {
      res = await api.POST('/auth/login', { body: { email, password } });
    } catch {
      throw new ApiError('Could not reach the server. Check your network.');
    }
    const tokens = unwrap(res);
    set({ accessToken: tokens.access_token });
    const user = unwrap(await api.GET('/me'));
    set({ user, status: 'signedIn' });
  },

  async signOut() {
    try {
      await api.POST('/auth/logout'); // clears the HttpOnly refresh-token cookie
    } catch {
      /* offline: still sign out locally */
    }
    set({ accessToken: null, user: null, status: 'signedOut' });
  },
}));

/** Exchanges the refresh-token cookie for a new access token. Single-flight. */
export function refreshAccessToken(): Promise<string | null> {
  refreshInFlight ??= (async () => {
    try {
      const res = await api.POST('/auth/refresh');
      if (res.error !== undefined || !res.data) return null;
      useSession.setState({ accessToken: res.data.access_token });
      return res.data.access_token;
    } catch {
      return null;
    } finally {
      refreshInFlight = null;
    }
  })();
  return refreshInFlight;
}

configureClient({
  getOrigin: () => useSession.getState().origin,
  getAccessToken: () => useSession.getState().accessToken,
  async refresh() {
    const token = await refreshAccessToken();
    if (!token) useSession.setState({ status: 'signedOut', accessToken: null, user: null });
    return token;
  },
});

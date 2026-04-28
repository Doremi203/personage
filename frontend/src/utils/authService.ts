import { ApiError, throwIfError } from './apiError';
import { clearNotificationsCache } from './notificationsCache';
import { clearUserCache } from './userCache';

const AUTH_API_URL =
  (import.meta.env.VITE_AUTH_API_URL as string | undefined) ??
  'https://auth.persomanage.ru';

const TOKENS_KEY = 'personage_auth_tokens';
export const AUTH_STATE_CHANGE_EVENT = 'personage-auth-state-change';

export const OAUTH_PROVIDER_STORAGE_KEY = 'personage_oauth_provider';
export type OAuthProvider = 'gmail' | 'google-calendar';

export type IntegrationType = 'Gmail' | 'GoogleCalendar' | 'Telegram';

export interface AuthTokens {
  accessToken: string;
  refreshToken?: string | null;
}

export interface UserApiResponse {
  email: string;
  name: string;
  gmailIntegration: {
    enabled: boolean;
    gmail?: string | null;
  };
  telegramIntegration: {
    enabled: boolean;
  };
  googleCalendarIntegration: {
    enabled: boolean;
    gmail?: string | null;
  };
}

function notifyAuthStateChanged(): void {
  window.dispatchEvent(new Event(AUTH_STATE_CHANGE_EVENT));
}

export function getTokens(): AuthTokens | null {
  const raw = localStorage.getItem(TOKENS_KEY);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as AuthTokens;
  } catch {
    return null;
  }
}

function setTokens(tokens: AuthTokens): void {
  localStorage.setItem(TOKENS_KEY, JSON.stringify(tokens));
  notifyAuthStateChanged();
}

function clearTokens(): void {
  localStorage.removeItem(TOKENS_KEY);
  clearUserCache();
  clearNotificationsCache();
  notifyAuthStateChanged();
}

export function isAuthenticated(): boolean {
  return getTokens() !== null;
}

export function logout(): void {
  clearTokens();
}

export async function login(
  email: string,
  password: string,
): Promise<AuthTokens> {
  const response = await fetch(
    `${AUTH_API_URL}/auth/personage/login/password`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    },
  );

  await throwIfError(response);

  const data = (await response.json()) as {
    accessToken: string;
    refreshToken?: string | null;
  };
  const tokens: AuthTokens = {
    accessToken: data.accessToken,
    refreshToken: data.refreshToken,
  };
  setTokens(tokens);
  return tokens;
}

export async function register(
  email: string,
  password: string,
  name: string,
): Promise<AuthTokens> {
  const response = await fetch(`${AUTH_API_URL}/auth/personage/register`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password, name }),
  });

  await throwIfError(response);

  const data = (await response.json()) as {
    accessToken: string;
    refreshToken?: string | null;
  };
  const tokens: AuthTokens = {
    accessToken: data.accessToken,
    refreshToken: data.refreshToken,
  };
  setTokens(tokens);
  return tokens;
}

export async function refreshAccessToken(): Promise<AuthTokens | null> {
  const tokens = getTokens();
  if (!tokens?.refreshToken) return null;

  const response = await fetch(`${AUTH_API_URL}/auth/personage/refresh`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refreshToken: tokens.refreshToken }),
  });

  if (!response.ok) {
    clearTokens();
    return null;
  }

  const data = (await response.json()) as {
    accessToken: string;
    refreshToken?: string | null;
  };
  const newTokens: AuthTokens = {
    accessToken: data.accessToken,
    refreshToken: data.refreshToken ?? tokens.refreshToken,
  };
  setTokens(newTokens);
  return newTokens;
}

export async function fetchWithTokenRefresh(
  doFetch: (accessToken: string) => Promise<Response>,
): Promise<Response> {
  const tokens = getTokens();
  if (!tokens) throw new ApiError(401, 'Необходимо войти в аккаунт.');

  let response = await doFetch(tokens.accessToken);
  if (response.status !== 401) {
    return response;
  }

  const newTokens = await refreshAccessToken();
  if (!newTokens) {
    clearTokens();
    throw new ApiError(401, 'Сессия истекла. Войдите заново.');
  }

  response = await doFetch(newTokens.accessToken);
  if (response.status === 401) {
    clearTokens();
    throw new ApiError(401, 'Сессия истекла. Войдите заново.');
  }

  return response;
}

export async function resetPassword(
  token: string,
  newPassword: string,
): Promise<AuthTokens> {
  const response = await fetch(`${AUTH_API_URL}/auth/personage/reset-password`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ token, newPassword }),
  });

  await throwIfError(response);

  const data = (await response.json()) as {
    accessToken: string;
    refreshToken?: string | null;
  };
  const tokens: AuthTokens = {
    accessToken: data.accessToken,
    refreshToken: data.refreshToken,
  };
  setTokens(tokens);
  return tokens;
}

export async function forgotPassword(
  email: string,
  resetUrlBase: string,
): Promise<void> {
  const response = await fetch(
    `${AUTH_API_URL}/auth/personage/forgot-password`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, resetUrlBase }),
    },
  );

  await throwIfError(response);
}

export async function startGmailAuth(
  userEmail: string,
  redirectUri: string,
): Promise<{ authorizationUrl: string; state: string }> {
  const response = await fetch(`${AUTH_API_URL}/auth/gmail/authorize`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ userEmail, redirectUri }),
  });

  await throwIfError(response);

  return (await response.json()) as { authorizationUrl: string; state: string };
}

export async function handleGmailCallback(
  userEmail: string,
  code: string,
  state: string,
  redirectUri: string,
): Promise<string> {
  const response = await fetch(`${AUTH_API_URL}/auth/gmail/callback`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ userEmail, code, state, redirectUri }),
  });

  await throwIfError(response);

  const data = (await response.json()) as { gmailEmail: string };
  return data.gmailEmail;
}

export async function startGoogleCalendarAuth(
  userEmail: string,
  redirectUri: string,
): Promise<{ authorizationUrl: string; state: string }> {
  const response = await fetchWithTokenRefresh((accessToken) =>
    fetch(`${AUTH_API_URL}/auth/google-calendar/authorize`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${accessToken}`,
      },
      body: JSON.stringify({ userEmail, redirectUri }),
    }),
  );

  await throwIfError(response);

  return (await response.json()) as { authorizationUrl: string; state: string };
}

export async function handleGoogleCalendarCallback(
  userEmail: string,
  code: string,
  state: string,
  redirectUri: string,
): Promise<string> {
  const response = await fetch(`${AUTH_API_URL}/auth/google-calendar/callback`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ userEmail, code, state, redirectUri }),
  });

  await throwIfError(response);

  const data = (await response.json()) as { gmailEmail: string };
  return data.gmailEmail;
}

export async function revokeIntegrationAccess(
  integrationType: IntegrationType,
): Promise<void> {
  const response = await fetchWithTokenRefresh((accessToken) =>
    fetch(`${AUTH_API_URL}/integrations/revoke-access`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${accessToken}`,
      },
      body: JSON.stringify({ integrationType }),
    }),
  );

  await throwIfError(response);
}

export async function fetchCurrentUser(): Promise<UserApiResponse> {
  const response = await fetchWithTokenRefresh((accessToken) =>
    fetch(`${AUTH_API_URL}/user`, {
      headers: { Authorization: `Bearer ${accessToken}` },
    }),
  );

  await throwIfError(response);

  return (await response.json()) as UserApiResponse;
}

export async function updateUserName(name: string): Promise<void> {
  const response = await fetchWithTokenRefresh((accessToken) =>
    fetch(`${AUTH_API_URL}/user`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${accessToken}`,
      },
      body: JSON.stringify({ name }),
    }),
  );

  if (!response.ok) {
    const error = await response.json().catch(() => ({}));
    throw new Error(
      (error as { message?: string }).message ??
        `Не удалось изменить имя: ${response.status}`,
    );
  }
}

export function getCurrentUserId(): string | null {
  const tokens = getTokens();
  if (!tokens?.accessToken) return null;
  const parts = tokens.accessToken.split('.');
  if (parts.length < 2) return null;
  const base64 = parts[1].replace(/-/g, '+').replace(/_/g, '/');
  const padded = base64.padEnd(base64.length + ((4 - (base64.length % 4)) % 4), '=');
  try {
    const bytes = Uint8Array.from(atob(padded), (c) => c.charCodeAt(0));
    const payload = JSON.parse(new TextDecoder().decode(bytes)) as { user_id?: string };
    return payload.user_id ?? null;
  } catch {
    return null;
  }
}

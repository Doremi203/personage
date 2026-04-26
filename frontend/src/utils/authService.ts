import { clearNotificationsCache } from './notificationsCache';
import { clearUserCache } from './userCache';

const AUTH_API_URL =
  (import.meta.env.VITE_AUTH_API_URL as string | undefined) ??
  'https://auth.persomanage.ru';

const TOKENS_KEY = 'personage_auth_tokens';
export const AUTH_STATE_CHANGE_EVENT = 'personage-auth-state-change';

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

  if (!response.ok) {
    const error = await response.json().catch(() => ({}));
    throw new Error(
      (error as { message?: string }).message ??
        `Ошибка входа: ${response.status}`,
    );
  }

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

  if (!response.ok) {
    const error = await response.json().catch(() => ({}));
    throw new Error(
      (error as { message?: string }).message ??
        `Ошибка регистрации: ${response.status}`,
    );
  }

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
  if (!tokens) throw new Error('Не авторизован');

  let response = await doFetch(tokens.accessToken);
  if (response.status !== 401) {
    return response;
  }

  const newTokens = await refreshAccessToken();
  if (!newTokens) {
    clearTokens();
    throw new Error('Сессия истекла');
  }

  response = await doFetch(newTokens.accessToken);
  if (response.status === 401) {
    clearTokens();
    throw new Error('Сессия истекла');
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

  if (!response.ok) {
    const error = await response.json().catch(() => ({}));
    throw new Error(
      (error as { message?: string }).message ??
        `Ошибка сброса пароля: ${response.status}`,
    );
  }

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

  if (!response.ok) {
    throw new Error(`Ошибка: ${response.status}`);
  }
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

  if (!response.ok) {
    const error = await response.json().catch(() => ({}));
    throw new Error(
      (error as { message?: string }).message ??
        `Ошибка запуска авторизации Gmail: ${response.status}`,
    );
  }

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

  if (!response.ok) {
    const error = await response.json().catch(() => ({}));
    throw new Error(
      (error as { message?: string }).message ??
        `Ошибка подключения Gmail: ${response.status}`,
    );
  }

  const data = (await response.json()) as { gmailEmail: string };
  return data.gmailEmail;
}

export async function fetchCurrentUser(): Promise<UserApiResponse> {
  const response = await fetchWithTokenRefresh((accessToken) =>
    fetch(`${AUTH_API_URL}/user`, {
      headers: { Authorization: `Bearer ${accessToken}` },
    }),
  );

  if (!response.ok) {
    const error = await response.json().catch(() => ({}));
    throw new Error(
      (error as { message?: string }).message ??
        `Ошибка получения данных пользователя: ${response.status}`,
    );
  }

  return (await response.json()) as UserApiResponse;
}

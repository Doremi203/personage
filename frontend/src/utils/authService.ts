const AUTH_API_URL =
  (import.meta.env.VITE_AUTH_API_URL as string | undefined) ??
  'https://auth.persomanage.ru';

const TOKENS_KEY = 'personage_auth_tokens';
const USER_INFO_KEY = 'personage_user_info';

export interface AuthTokens {
  accessToken: string;
  refreshToken?: string | null;
}

export interface UserInfo {
  email: string;
  name?: string;
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

export function setTokens(tokens: AuthTokens): void {
  localStorage.setItem(TOKENS_KEY, JSON.stringify(tokens));
}

export function clearTokens(): void {
  localStorage.removeItem(TOKENS_KEY);
  localStorage.removeItem(USER_INFO_KEY);
}

export function getUserInfo(): UserInfo | null {
  const raw = localStorage.getItem(USER_INFO_KEY);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as UserInfo;
  } catch {
    return null;
  }
}

export function setUserInfo(info: UserInfo): void {
  localStorage.setItem(USER_INFO_KEY, JSON.stringify(info));
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
  setUserInfo({ email });
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
  setUserInfo({ email, name });
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

const TELEGRAM_AUTH_API_URL =
  (import.meta.env.VITE_TELEGRAM_AUTH_API_URL as string | undefined) ??
  'https://telegram-auth.persomanage.ru';

export type TelegramAuthMethod = 'qr' | 'phone';

export interface InitiateAuthResponse {
  login_id: string;
  method: TelegramAuthMethod;
  qr_data?: string | null;
  expires_in: number;
  phone_code_hash?: string | null;
}

export type VerifyStatus = 'success' | 'password_required' | 'error';

export interface VerifyCodeResponse {
  status: VerifyStatus;
  message?: string | null;
}

export type AuthStatus = 'pending' | 'completed' | 'expired' | 'failed';

export interface AuthStatusResponse {
  status: AuthStatus;
  user_id?: string | null;
  error?: string | null;
}

async function postJson<T>(path: string, body: unknown): Promise<T> {
  const response = await fetch(`${TELEGRAM_AUTH_API_URL}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  if (!response.ok) {
    const error = await response.json().catch(() => ({}));
    throw new Error(
      (error as { detail?: string; message?: string }).detail ??
        (error as { detail?: string; message?: string }).message ??
        `Ошибка Telegram: ${response.status}`,
    );
  }
  return (await response.json()) as T;
}

export async function initiateTelegramAuth(
  userId: string,
  method: TelegramAuthMethod,
  phone?: string,
): Promise<InitiateAuthResponse> {
  return postJson<InitiateAuthResponse>('/v1/auth/initiate', {
    user_id: userId,
    method,
    ...(phone ? { phone } : {}),
  });
}

export async function verifyTelegramCode(
  loginId: string,
  code: string,
  password?: string,
): Promise<VerifyCodeResponse> {
  return postJson<VerifyCodeResponse>('/v1/auth/verify', {
    login_id: loginId,
    code,
    ...(password ? { password } : {}),
  });
}

export async function resendTelegramCode(loginId: string): Promise<void> {
  await postJson('/v1/auth/resend-code', { login_id: loginId });
}

export async function getTelegramAuthStatus(loginId: string): Promise<AuthStatusResponse> {
  const response = await fetch(
    `${TELEGRAM_AUTH_API_URL}/v1/auth/status/${encodeURIComponent(loginId)}`,
  );
  if (!response.ok) {
    throw new Error(`Ошибка статуса Telegram: ${response.status}`);
  }
  return (await response.json()) as AuthStatusResponse;
}

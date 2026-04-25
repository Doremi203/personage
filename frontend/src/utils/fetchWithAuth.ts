import { fetchWithTokenRefresh } from './authService';

export async function fetchWithAuth<T>(
  url: string,
  options: RequestInit = {},
): Promise<T> {
  const response = await fetchWithTokenRefresh((accessToken) =>
    fetch(url, {
      ...options,
      headers: {
        ...options.headers,
        'User-Token': accessToken,
      },
    }),
  );

  if (!response.ok) {
    const error = await response.json().catch(() => ({}));
    throw new Error(
      (error as { message?: string }).message ?? `Ошибка: ${response.status}`,
    );
  }

  const text = await response.text();
  return text ? (JSON.parse(text) as T) : ({} as T);
}

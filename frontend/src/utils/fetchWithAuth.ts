import { throwIfError } from './apiError';
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

  await throwIfError(response);

  const text = await response.text();
  return text ? (JSON.parse(text) as T) : ({} as T);
}

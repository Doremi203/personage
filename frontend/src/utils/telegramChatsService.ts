import { throwIfError } from './apiError';
import { fetchWithTokenRefresh } from './authService';

const AUTH_API_URL =
  (import.meta.env.VITE_AUTH_API_URL as string | undefined) ??
  'https://auth.persomanage.ru';

export interface TelegramChat {
  chatId: number;
  chatName: string;
  isActive: boolean;
}

interface GetUserChatsResponse {
  chats: TelegramChat[];
}

export async function getUserChats(): Promise<TelegramChat[]> {
  const response = await fetchWithTokenRefresh((accessToken) =>
    fetch(`${AUTH_API_URL}/get-user-chats`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${accessToken}` },
    }),
  );

  await throwIfError(response);

  const data = (await response.json()) as GetUserChatsResponse;
  return data.chats ?? [];
}

export async function updateUserChat(
  chatId: number,
  isActive: boolean,
): Promise<void> {
  const response = await fetchWithTokenRefresh((accessToken) =>
    fetch(`${AUTH_API_URL}/update-user-chat`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${accessToken}`,
      },
      body: JSON.stringify({ chatId, isActive }),
    }),
  );

  await throwIfError(response);
}

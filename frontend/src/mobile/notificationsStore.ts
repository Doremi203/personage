import { useEffect, useState } from 'react';
import {
  listNotifications,
  markAllNotificationsRead,
  markNotificationRead,
  type ApiNotificationItem,
} from '../utils/notificatorService';
import { clearNotificationsCache } from '../utils/notificationsCache';

const LEGACY_READ_KEY = 'personage_notifications_read';
const PAGE_SIZE = 10;
const MAX_PAGES = 10;
const BACKOFF_BASE_MS = 1_000;
const BACKOFF_MAX_MS = 60_000;

try { localStorage.removeItem(LEGACY_READ_KEY); } catch { /* best-effort */ }

let items: ApiNotificationItem[] = [];
let loading = false;
let loaded = false;
let error: string | null = null;
let failures = 0;
let nextRetryAt = 0;
const listeners = new Set<() => void>();

function emit(): void {
  listeners.forEach((cb) => cb());
}

export interface NotificationsState {
  items: ApiNotificationItem[];
  read: Set<string>;
  unreadCount: number;
  loading: boolean;
  loaded: boolean;
  error: string | null;
}

function snapshot(): NotificationsState {
  const read = new Set(items.filter((i) => i.readAt).map((i) => i.id));
  return {
    items,
    read,
    unreadCount: items.length - read.size,
    loading,
    loaded,
    error,
  };
}

export async function refreshNotifications(opts: { force?: boolean } = {}): Promise<void> {
  if (loading) return;
  if (!opts.force && Date.now() < nextRetryAt) return;
  loading = true;
  error = null;
  emit();
  try {
    const collected: ApiNotificationItem[] = [];
    for (let page = 1; page <= MAX_PAGES; page++) {
      const data = await listNotifications(page, PAGE_SIZE);
      const batch = data.notifications ?? [];
      collected.push(...batch);
      if (batch.length < PAGE_SIZE) break;
    }
    items = collected;
    loaded = true;
    failures = 0;
    nextRetryAt = 0;
  } catch (err) {
    error = err instanceof Error ? err.message : 'Не удалось загрузить уведомления';
    failures++;
    const delay = Math.min(BACKOFF_BASE_MS * 2 ** (failures - 1), BACKOFF_MAX_MS);
    nextRetryAt = Date.now() + delay;
  } finally {
    loading = false;
    emit();
  }
}

export async function markRead(id: string): Promise<void> {
  const idx = items.findIndex((it) => it.id === id);
  if (idx < 0 || items[idx].readAt) return;
  const prev = items[idx];
  const optimistic = { ...prev, readAt: new Date().toISOString() };
  items = items.map((it, i) => (i === idx ? optimistic : it));
  emit();
  try {
    await markNotificationRead(id);
    clearNotificationsCache();
  } catch {
    items = items.map((it, i) => (i === idx ? prev : it));
    emit();
  }
}

export async function markAllRead(): Promise<void> {
  if (items.length === 0) return;
  const prev = items;
  const now = new Date().toISOString();
  items = items.map((it) => (it.readAt ? it : { ...it, readAt: now }));
  emit();
  try {
    await markAllNotificationsRead();
    clearNotificationsCache();
  } catch {
    items = prev;
    emit();
  }
}

export function useNotifications(): NotificationsState {
  const [, force] = useState({});
  useEffect(() => {
    const cb = () => force({});
    listeners.add(cb);
    return () => { listeners.delete(cb); };
  }, []);
  return snapshot();
}

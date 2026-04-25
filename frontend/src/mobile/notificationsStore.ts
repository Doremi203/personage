import { useEffect, useState } from 'react';
import { listNotifications, type ApiNotificationItem } from '../utils/notificatorService';

const READ_KEY = 'personage_notifications_read';
const PAGE_SIZE = 10;
const MAX_PAGES = 10;
const BACKOFF_BASE_MS = 1_000;
const BACKOFF_MAX_MS = 60_000;

let items: ApiNotificationItem[] = [];
let read: Set<string> = loadReadSet();
let loading = false;
let loaded = false;
let error: string | null = null;
let failures = 0;
let nextRetryAt = 0;
const listeners = new Set<() => void>();

function loadReadSet(): Set<string> {
  try {
    const raw = localStorage.getItem(READ_KEY);
    if (!raw) return new Set();
    const arr = JSON.parse(raw) as unknown;
    if (!Array.isArray(arr)) return new Set();
    return new Set(arr.filter((x): x is string => typeof x === 'string'));
  } catch {
    return new Set();
  }
}

function persistReadSet(): void {
  try {
    localStorage.setItem(READ_KEY, JSON.stringify(Array.from(read)));
  } catch {
    // best-effort
  }
}

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
  return {
    items,
    read,
    unreadCount: items.reduce((acc, it) => acc + (read.has(it.id) ? 0 : 1), 0),
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

export function markRead(id: string): void {
  if (read.has(id)) return;
  read = new Set(read).add(id);
  persistReadSet();
  emit();
}

export function markAllRead(): void {
  if (items.length === 0) return;
  const next = new Set(read);
  for (const it of items) next.add(it.id);
  read = next;
  persistReadSet();
  emit();
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

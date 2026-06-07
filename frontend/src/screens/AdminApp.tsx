import { useCallback, useEffect, useState } from 'react';
import { ADMIN_KEY_CHANGE_EVENT, getAdminKey } from '../utils/adminService';
import { AdminLoginScreen } from './AdminLoginScreen';
import { AdminUsersScreen } from './AdminUsersScreen';
import { AdminUserTasksScreen, type AdminUserTab } from './AdminUserTasksScreen';
import { AdminPromptsScreen } from './AdminPromptsScreen';
import { AdminGenerationSettingsScreen } from './AdminGenerationSettingsScreen';

interface AdminRoute {
  view: 'users' | 'user' | 'prompts' | 'generation-settings';
  userId?: string;
  tab?: AdminUserTab;
}

const USER_PATH_RE = /^\/admin\/users\/([^/]+)(?:\/(tasks|clusters|events|notifications))?\/?$/;

function parseLocation(): AdminRoute {
  const path = window.location.pathname;
  if (path === '/admin/prompts' || path === '/admin/prompts/') {
    return { view: 'prompts' };
  }
  if (path === '/admin/generation-settings' || path === '/admin/generation-settings/') {
    return { view: 'generation-settings' };
  }
  const match = USER_PATH_RE.exec(path);
  if (match) {
    return {
      view: 'user',
      userId: decodeURIComponent(match[1]),
      tab: (match[2] as AdminUserTab | undefined) ?? 'tasks',
    };
  }
  return { view: 'users' };
}

export function AdminApp() {
  const [hasKey, setHasKey] = useState(() => getAdminKey() !== null);
  const [route, setRoute] = useState<AdminRoute>(() => parseLocation());

  useEffect(() => {
    const syncKey = () => setHasKey(getAdminKey() !== null);
    window.addEventListener(ADMIN_KEY_CHANGE_EVENT, syncKey);
    window.addEventListener('storage', syncKey);
    return () => {
      window.removeEventListener(ADMIN_KEY_CHANGE_EVENT, syncKey);
      window.removeEventListener('storage', syncKey);
    };
  }, []);

  useEffect(() => {
    const syncPath = () => setRoute(parseLocation());
    window.addEventListener('popstate', syncPath);
    return () => window.removeEventListener('popstate', syncPath);
  }, []);

  const goToUser = useCallback((userId: string, tab: AdminUserTab = 'tasks') => {
    const path = `/admin/users/${encodeURIComponent(userId)}/${tab}`;
    window.history.pushState({}, '', path);
    setRoute({ view: 'user', userId, tab });
  }, []);

  const goToUsers = useCallback(() => {
    window.history.pushState({}, '', '/admin');
    setRoute({ view: 'users' });
  }, []);

  const goToPrompts = useCallback(() => {
    window.history.pushState({}, '', '/admin/prompts');
    setRoute({ view: 'prompts' });
  }, []);

  const goToGenerationSettings = useCallback(() => {
    window.history.pushState({}, '', '/admin/generation-settings');
    setRoute({ view: 'generation-settings' });
  }, []);

  if (!hasKey) {
    return <AdminLoginScreen />;
  }

  if (route.view === 'prompts') {
    return <AdminPromptsScreen onBack={goToUsers} />;
  }

  if (route.view === 'generation-settings') {
    return <AdminGenerationSettingsScreen onBack={goToUsers} />;
  }

  if (route.view === 'user' && route.userId) {
    return (
      <AdminUserTasksScreen
        userId={route.userId}
        activeTab={route.tab ?? 'tasks'}
        onBack={goToUsers}
        onChangeTab={(tab) => goToUser(route.userId!, tab)}
      />
    );
  }

  return (
    <AdminUsersScreen
      onSelect={(userId) => goToUser(userId)}
      onOpenPrompts={goToPrompts}
      onOpenGenerationSettings={goToGenerationSettings}
    />
  );
}

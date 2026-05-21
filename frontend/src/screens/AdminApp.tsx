import { useCallback, useEffect, useState } from 'react';
import { ADMIN_KEY_CHANGE_EVENT, getAdminKey } from '../utils/adminService';
import { AdminLoginScreen } from './AdminLoginScreen';
import { AdminUsersScreen } from './AdminUsersScreen';
import { AdminUserTasksScreen } from './AdminUserTasksScreen';

const USER_PATH_RE = /^\/admin\/users\/([^/]+)/;

function pathSelectedUserId(): string | null {
  const match = USER_PATH_RE.exec(window.location.pathname);
  return match ? decodeURIComponent(match[1]) : null;
}

export function AdminApp() {
  const [hasKey, setHasKey] = useState(() => getAdminKey() !== null);
  const [selectedUserId, setSelectedUserId] = useState<string | null>(() => pathSelectedUserId());

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
    const syncPath = () => setSelectedUserId(pathSelectedUserId());
    window.addEventListener('popstate', syncPath);
    return () => window.removeEventListener('popstate', syncPath);
  }, []);

  const goToUser = useCallback((userId: string) => {
    window.history.pushState({}, '', `/admin/users/${encodeURIComponent(userId)}`);
    setSelectedUserId(userId);
  }, []);

  const goToUsers = useCallback(() => {
    window.history.pushState({}, '', '/admin');
    setSelectedUserId(null);
  }, []);

  if (!hasKey) {
    return <AdminLoginScreen />;
  }

  if (selectedUserId) {
    return <AdminUserTasksScreen userId={selectedUserId} onBack={goToUsers} />;
  }

  return <AdminUsersScreen onSelect={goToUser} />;
}

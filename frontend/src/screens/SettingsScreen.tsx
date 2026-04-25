import { useEffect, useState } from 'react';
import {
  BarChart2,
  Bell,
  CalendarClock,
  CheckCircle2,
  ChevronRight,
  Loader2,
  Mail,
  Send,
  type LucideIcon,
} from 'lucide-react';
import { LargeHeader } from '../mobile/Chrome';
import { SANS, SERIF, T } from '../mobile/tokens';
import {
  OAUTH_PROVIDER_STORAGE_KEY,
  fetchCurrentUser,
  startGmailAuth,
  startGoogleCalendarAuth,
  type UserApiResponse,
} from '../utils/authService';
import { clearUserCache } from '../utils/userCache';
import {
  getNotificationSettings,
  toggleNotification,
} from '../utils/notificatorService';

interface SettingsScreenProps {
  onLogout: () => void;
}

interface NotifMeta {
  title: string;
  subtitle?: string;
  Icon: LucideIcon;
  bg: string;
  ink: string;
}

const NOTIF_META: Record<string, NotifMeta> = {
  upcoming_event: {
    title: 'Напоминания о задачах',
    subtitle: 'За час до дедлайна',
    Icon: Bell,
    bg: T.amberFill,
    ink: T.amberDp,
  },
  schedule_change: {
    title: 'Изменения в расписании',
    Icon: CalendarClock,
    bg: T.infoFill,
    ink: T.info,
  },
};

function notifMeta(type: string): NotifMeta {
  return NOTIF_META[type] ?? {
    title: type,
    Icon: BarChart2,
    bg: T.okFill,
    ink: T.ok,
  };
}

interface NotifRow {
  type: string;
  enabled: boolean;
  toggling: boolean;
}

const SettingsScreen = ({ onLogout }: SettingsScreenProps) => {
  const [settings, setSettings] = useState<NotifRow[]>([]);
  const [user, setUser] = useState<UserApiResponse | null>(null);
  const [userLoading, setUserLoading] = useState(true);
  const [gmail, setGmail] = useState<string | null>(null);
  const [gmailLoading, setGmailLoading] = useState(false);
  const [gmailError, setGmailError] = useState<string | null>(null);
  const [calendar, setCalendar] = useState<string | null>(null);
  const [calendarLoading, setCalendarLoading] = useState(false);
  const [calendarError, setCalendarError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const { settings } = await getNotificationSettings();
        if (!cancelled) {
          setSettings(settings.map((s) => ({ ...s, toggling: false })));
        }
      } catch (err) {
        console.error('Failed to fetch notification settings:', err);
      }
    })();
    return () => { cancelled = true; };
  }, []);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const data = await fetchCurrentUser();
        if (cancelled) return;
        setUser(data);
        if (data.gmailIntegration.enabled && data.gmailIntegration.gmail) {
          setGmail(data.gmailIntegration.gmail);
        }
        if (data.googleCalendarIntegration.enabled && data.googleCalendarIntegration.gmail) {
          setCalendar(data.googleCalendarIntegration.gmail);
        }
      } catch (err) {
        console.error('Failed to fetch user data:', err);
      } finally {
        if (!cancelled) setUserLoading(false);
      }
    })();
    return () => { cancelled = true; };
  }, []);

  const handleConnectGmail = async () => {
    setGmailError(null);
    setGmailLoading(true);
    try {
      if (!user?.email) {
        setGmailError('Не удалось определить email пользователя');
        return;
      }
      sessionStorage.setItem(OAUTH_PROVIDER_STORAGE_KEY, 'gmail');
      const { authorizationUrl } = await startGmailAuth(user.email, window.location.origin);
      window.location.href = authorizationUrl;
    } catch (err) {
      setGmailError(err instanceof Error ? err.message : 'Не удалось запустить авторизацию Gmail');
      setGmailLoading(false);
    }
  };

  const handleDisconnectGmail = () => {
    clearUserCache();
    setGmail(null);
    setGmailError(null);
  };

  const handleConnectCalendar = async () => {
    setCalendarError(null);
    setCalendarLoading(true);
    try {
      sessionStorage.setItem(OAUTH_PROVIDER_STORAGE_KEY, 'google-calendar');
      const { authorizationUrl } = await startGoogleCalendarAuth(window.location.origin);
      window.location.href = authorizationUrl;
    } catch (err) {
      setCalendarError(err instanceof Error ? err.message : 'Не удалось запустить авторизацию Google Calendar');
      setCalendarLoading(false);
    }
  };

  const handleDisconnectCalendar = () => {
    clearUserCache();
    setCalendar(null);
    setCalendarError(null);
  };

  const handleToggle = (type: string) => {
    setSettings((prev) => prev.map((s) => s.type === type ? { ...s, toggling: true } : s));
    void toggleNotification(type)
      .then((enabled) => {
        setSettings((prev) => prev.map((s) =>
          s.type === type ? { ...s, enabled, toggling: false } : s,
        ));
      })
      .catch((err: unknown) => {
        console.error('Failed to toggle notification setting:', err);
        setSettings((prev) => prev.map((s) =>
          s.type === type ? { ...s, toggling: false } : s,
        ));
      });
  };

  const displayName  = userLoading ? 'Загрузка…' : (user?.name  ?? 'Пользователь');
  const displayEmail = userLoading ? ''           : (user?.email ?? '');

  const telegramConnected = user?.telegramIntegration.enabled ?? false;

  return (
    <>
      <LargeHeader title="Настройки" />

      {/* Profile card */}
      <div style={{ padding: '4px 16px 18px' }}>
        <div style={{
          background: T.surface, borderRadius: 14,
          border: `0.5px solid ${T.hairline}`,
          padding: '14px 16px',
          display: 'flex', alignItems: 'center', gap: 14,
        }}>
          <div style={{
            width: 52, height: 52, borderRadius: '50%',
            background: `linear-gradient(135deg, ${T.amber}, ${T.amberDp})`,
            color: T.ink,
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            fontFamily: SERIF, fontSize: 22, fontWeight: 600,
            flexShrink: 0,
          }}>
            {(displayName || 'П').charAt(0).toUpperCase()}
          </div>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{
              fontFamily: SERIF, fontSize: 19, color: T.ink,
              letterSpacing: -0.2, lineHeight: 1.2,
              overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
            }}>{displayName}</div>
            <div style={{
              fontSize: 13, color: T.ink3, marginTop: 2,
              overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
            }}>{displayEmail}</div>
          </div>
          <ChevronRight size={18} strokeWidth={1.8} style={{ color: T.ink4 }} />
        </div>
      </div>

      {/* Notifications */}
      {settings.length > 0 && (
        <SetGroup label="Уведомления">
          {settings.map((s, i) => {
            const meta = notifMeta(s.type);
            return (
              <ToggleRow
                key={s.type}
                icon={meta.Icon}
                iconBg={meta.bg}
                iconInk={meta.ink}
                title={meta.title}
                subtitle={meta.subtitle}
                value={s.enabled}
                disabled={s.toggling}
                onChange={() => handleToggle(s.type)}
                last={i === settings.length - 1}
              />
            );
          })}
        </SetGroup>
      )}

      {/* Sources */}
      <SetGroup label="Источники задач">
        {gmailError && (
          <div style={{
            padding: '10px 14px',
            fontSize: 13, color: T.danger, background: T.dangerFill,
            borderBottom: `0.5px solid ${T.hairline}`,
          }}>{gmailError}</div>
        )}
        {gmail ? (
          <NavRow
            icon={Mail}
            iconBg={T.dangerFill}
            iconInk={T.danger}
            title="Gmail"
            value={gmail}
            actionLabel="Отключить"
            onAction={handleDisconnectGmail}
          />
        ) : (
          <ActionRow
            icon={Mail}
            iconBg={T.dangerFill}
            iconInk={T.danger}
            title="Gmail"
            subtitle="Подключить почту"
            actionLabel={gmailLoading ? 'Переход…' : 'Подключить'}
            disabled={gmailLoading}
            loading={gmailLoading}
            onAction={() => void handleConnectGmail()}
          />
        )}
        {calendarError && (
          <div style={{
            padding: '10px 14px',
            fontSize: 13, color: T.danger, background: T.dangerFill,
            borderBottom: `0.5px solid ${T.hairline}`,
          }}>{calendarError}</div>
        )}
        {calendar ? (
          <NavRow
            icon={CalendarClock}
            iconBg={T.okFill}
            iconInk={T.ok}
            title="Google Calendar"
            value={calendar}
            actionLabel="Отключить"
            onAction={handleDisconnectCalendar}
          />
        ) : (
          <ActionRow
            icon={CalendarClock}
            iconBg={T.okFill}
            iconInk={T.ok}
            title="Google Calendar"
            subtitle="Подключить календарь"
            actionLabel={calendarLoading ? 'Переход…' : 'Подключить'}
            disabled={calendarLoading}
            loading={calendarLoading}
            onAction={() => void handleConnectCalendar()}
          />
        )}
        <ReadonlyRow
          icon={Send}
          iconBg={T.infoFill}
          iconInk={T.info}
          title="Telegram"
          value={userLoading ? 'Загрузка…' : (telegramConnected ? 'Подключён' : 'Не подключён')}
          last
        />
      </SetGroup>

      {/* Logout */}
      <div style={{ padding: '0 16px 24px' }}>
        <button
          type="button"
          onClick={onLogout}
          style={{
            width: '100%', padding: 14,
            background: T.surface, color: T.danger,
            border: `0.5px solid ${T.hairline}`,
            borderRadius: 14, cursor: 'pointer',
            fontFamily: SANS, fontSize: 15, fontWeight: 500,
          }}
        >
          Выйти из аккаунта
        </button>
      </div>
    </>
  );
};

function SetGroup({ label, children }: { label?: string; children: React.ReactNode }) {
  return (
    <div style={{ marginBottom: 22 }}>
      {label && (
        <div style={{
          padding: '0 24px 6px',
          fontSize: 11.5, color: T.ink3, fontWeight: 600,
          letterSpacing: 0.3, textTransform: 'uppercase',
        }}>{label}</div>
      )}
      <div style={{ padding: '0 16px' }}>
        <div style={{
          background: T.surface, borderRadius: 14,
          border: `0.5px solid ${T.hairline}`,
          overflow: 'hidden',
        }}>{children}</div>
      </div>
    </div>
  );
}

interface RowFrameProps {
  icon: LucideIcon;
  iconBg: string;
  iconInk: string;
  title: string;
  subtitle?: string;
  last?: boolean;
  children?: React.ReactNode;
}

function RowFrame({ icon: Icon, iconBg, iconInk, title, subtitle, last, children }: RowFrameProps) {
  return (
    <div style={{
      display: 'flex', alignItems: 'center', gap: 12,
      padding: '11px 14px',
      borderBottom: last ? 'none' : `0.5px solid ${T.hairline}`,
    }}>
      <div style={{
        width: 30, height: 30, borderRadius: 8, background: iconBg, color: iconInk,
        display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0,
      }}>
        <Icon size={15} strokeWidth={1.8} />
      </div>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{
          fontSize: 14.5, color: T.ink, fontWeight: 500, lineHeight: 1.25,
          overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
        }}>{title}</div>
        {subtitle && (
          <div style={{
            fontSize: 12, color: T.ink3, marginTop: 1,
            overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
          }}>{subtitle}</div>
        )}
      </div>
      {children}
    </div>
  );
}

interface ToggleRowProps extends Omit<RowFrameProps, 'children'> {
  value: boolean;
  disabled?: boolean;
  onChange: (v: boolean) => void;
}

function ToggleRow({ value, disabled, onChange, ...rest }: ToggleRowProps) {
  return (
    <RowFrame {...rest}>
      <Toggle value={value} disabled={disabled} onChange={onChange} ariaLabel={rest.title} />
    </RowFrame>
  );
}

interface NavRowProps extends Omit<RowFrameProps, 'children'> {
  value?: string;
  actionLabel?: string;
  onAction?: () => void;
}

function NavRow({ value, actionLabel, onAction, ...rest }: NavRowProps) {
  return (
    <RowFrame {...rest}>
      {value && (
        <span style={{
          fontSize: 13, color: T.ink3, marginRight: actionLabel ? 6 : 4,
          maxWidth: 160, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
        }}>{value}</span>
      )}
      {actionLabel ? (
        <button
          type="button"
          onClick={onAction}
          style={{
            background: 'transparent', border: 'none', cursor: 'pointer',
            color: T.amberDp, fontSize: 13, fontWeight: 500,
            fontFamily: SANS, padding: '4px 2px',
          }}
        >{actionLabel}</button>
      ) : (
        <ChevronRight size={16} strokeWidth={1.8} style={{ color: T.ink4 }} />
      )}
    </RowFrame>
  );
}

interface ReadonlyRowProps extends Omit<RowFrameProps, 'children'> {
  value: string;
}

function ReadonlyRow({ value, ...rest }: ReadonlyRowProps) {
  return (
    <RowFrame {...rest}>
      {value === 'Подключён' && (
        <CheckCircle2 size={16} strokeWidth={1.8} style={{ color: T.ok, marginRight: 6 }} />
      )}
      <span style={{ fontSize: 13, color: T.ink3 }}>{value}</span>
    </RowFrame>
  );
}

interface ActionRowProps extends Omit<RowFrameProps, 'children'> {
  actionLabel: string;
  disabled?: boolean;
  loading?: boolean;
  onAction: () => void;
}

function ActionRow({ actionLabel, disabled, loading, onAction, ...rest }: ActionRowProps) {
  return (
    <RowFrame {...rest}>
      <button
        type="button"
        onClick={onAction}
        disabled={disabled}
        style={{
          background: T.amberDp, color: '#fff',
          border: 'none', cursor: disabled ? 'default' : 'pointer',
          padding: '6px 12px', borderRadius: 999,
          fontSize: 13, fontWeight: 600, fontFamily: SANS,
          display: 'inline-flex', alignItems: 'center', gap: 6,
          opacity: disabled ? 0.6 : 1,
        }}
      >
        {loading && <Loader2 size={13} className="animate-spin" />}
        {actionLabel}
      </button>
    </RowFrame>
  );
}

interface ToggleProps {
  value: boolean;
  disabled?: boolean;
  onChange: (v: boolean) => void;
  ariaLabel?: string;
}

function Toggle({ value, disabled, onChange, ariaLabel }: ToggleProps) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={value}
      aria-label={ariaLabel}
      onClick={() => !disabled && onChange(!value)}
      disabled={disabled}
      style={{
        width: 50, height: 30, borderRadius: 999,
        background: value ? T.amberDp : T.subtleHi,
        border: 'none', cursor: disabled ? 'default' : 'pointer',
        position: 'relative',
        transition: 'background .2s', flexShrink: 0,
        opacity: disabled ? 0.6 : 1,
      }}
    >
      <span style={{
        position: 'absolute', top: 2, left: value ? 22 : 2,
        width: 26, height: 26, borderRadius: '50%',
        background: '#fff',
        boxShadow: '0 2px 4px rgba(0,0,0,0.18), 0 0 0 0.5px rgba(0,0,0,0.04)',
        transition: 'left .2s',
      }} />
    </button>
  );
}

export default SettingsScreen;

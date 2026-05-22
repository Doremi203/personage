import { useState, type ReactNode } from 'react';
import {
  Bell,
  BellOff,
  Check,
  Loader2,
  SquarePlus,
  type LucideIcon,
} from 'lucide-react';
import { SANS, SERIF, T } from './tokens';
import {
  getNotificationPermission,
  isIos,
  isPushSupported,
  isStandalonePWA,
  setupPushNotifications,
} from '../utils/pushNotifications';

const IOS_INSTALL_DISMISSED_KEY = 'personage_ios_install_dismissed';
const PUSH_DISMISSED_KEY = 'personage_push_dismissed';

type Mode = 'ios-install' | 'push' | 'none';
type PushState = 'idle' | 'loading' | 'granted' | 'denied' | 'unsupported' | 'error';

function pickMode(authenticated: boolean): Mode {
  const iosInstallEligible =
    isIos() &&
    !isStandalonePWA() &&
    localStorage.getItem(IOS_INSTALL_DISMISSED_KEY) !== 'true';
  if (iosInstallEligible) return 'ios-install';

  if (!authenticated) return 'none';
  if (localStorage.getItem(PUSH_DISMISSED_KEY) === 'true') return 'none';
  if (!isPushSupported()) return 'none';
  const perm = getNotificationPermission();
  if (perm === 'unsupported' || perm === 'granted' || perm === 'denied') return 'none';
  return 'push';
}

interface OnboardingPromptProps {
  authenticated: boolean;
  onDismiss?: () => void;
}

export function OnboardingPrompt({ authenticated, onDismiss }: OnboardingPromptProps) {
  const [mode, setMode] = useState<Mode>(() => pickMode(authenticated));
  const [trackedAuthenticated, setTrackedAuthenticated] = useState(authenticated);
  const [pushState, setPushState] = useState<PushState>('idle');
  const [pushError, setPushError] = useState<string | null>(null);

  if (authenticated !== trackedAuthenticated) {
    setTrackedAuthenticated(authenticated);
    if (mode === 'none') setMode(pickMode(authenticated));
  }

  if (mode === 'none') return null;

  const dismiss = () => {
    if (mode === 'ios-install') {
      localStorage.setItem(IOS_INSTALL_DISMISSED_KEY, 'true');
    } else if (mode === 'push') {
      localStorage.setItem(PUSH_DISMISSED_KEY, 'true');
    }
    setMode(pickMode(authenticated));
    onDismiss?.();
  };

  const handleEnablePush = async () => {
    setPushState('loading');
    setPushError(null);
    const result = await setupPushNotifications();
    switch (result.status) {
      case 'subscribed':
        setPushState('granted');
        setTimeout(dismiss, 1200);
        break;
      case 'denied':
        setPushState('denied');
        break;
      case 'unsupported':
        setPushState('unsupported');
        break;
      case 'error':
        setPushError(result.error.message);
        setPushState('error');
        break;
    }
  };

  return (
    <div
      className="animate-consent-fade"
      style={{
        position: 'absolute', inset: 0, zIndex: 220,
        background: 'rgba(20,16,8,0.45)',
        backdropFilter: 'blur(2px)',
        display: 'flex', alignItems: 'flex-end', justifyContent: 'center',
        fontFamily: SANS,
      }}
    >
      <div
        className="animate-consent-rise"
        style={{
          width: '100%', background: T.bg, color: T.ink,
          borderRadius: '20px 20px 0 0',
          padding: '12px 20px 22px',
          boxShadow: '0 -10px 32px rgba(0,0,0,0.18)',
        }}
      >
        <div style={{
          width: 36, height: 5, borderRadius: 99,
          background: T.subtleHi, margin: '0 auto 12px',
        }} />

        {mode === 'ios-install'
          ? <IosInstallContent onDismiss={dismiss} />
          : <PushContent
              state={pushState}
              errorMessage={pushError}
              onEnable={() => void handleEnablePush()}
              onDismiss={dismiss}
            />}
      </div>
    </div>
  );
}

function IosInstallContent({ onDismiss }: { onDismiss: () => void }) {
  return (
    <>
      <PromptHeader
        title="Установить Personage"
        subtitle="Откроется как обычное приложение"
      />

      <div style={{
        background: T.surface, borderRadius: 12,
        border: `0.5px solid ${T.hairline}`,
        overflow: 'hidden', marginBottom: 12,
      }}>
        <PromptStep
          n={1}
          title={
            <>
              Нажмите{' '}
              <span style={{ color: T.amberDp, fontWeight: 600 }}>Поделиться</span>{' '}
              в Safari
            </>
          }
          visual={
            <div style={visualBox(T.amberFill, T.amberDp)}>
              <svg width="14" height="17" viewBox="0 0 18 22" fill="none" aria-hidden>
                <path d="M9 1L9 14M9 1L5 5M9 1L13 5"
                  stroke="currentColor" strokeWidth="2"
                  strokeLinecap="round" strokeLinejoin="round" />
                <path d="M3 9V19a2 2 0 002 2h8a2 2 0 002-2V9"
                  stroke="currentColor" strokeWidth="2"
                  strokeLinecap="round" strokeLinejoin="round" />
              </svg>
            </div>
          }
        />
        <PromptStep
          n={2}
          title={
            <>
              Выберите{' '}
              <span style={{ color: T.amberDp, fontWeight: 600 }}>«На экран „Домой"»</span>
            </>
          }
          visual={
            <div style={visualBox(T.infoFill, T.info)}>
              <SquarePlus size={16} strokeWidth={1.9} />
            </div>
          }
        />
        <PromptStep
          n={3}
          title="Нажмите «Добавить»"
          visual={
            <div style={visualBox(T.okFill, T.ok)}>
              <Check size={16} strokeWidth={2.4} />
            </div>
          }
          last
        />
      </div>

      <div style={{ display: 'flex', gap: 10 }}>
        <button type="button" onClick={onDismiss} style={secondaryBtn}>
          Не сейчас
        </button>
        <button type="button" onClick={onDismiss} style={primaryBtn}>
          <Check size={15} strokeWidth={2.2} />
          Понятно
        </button>
      </div>
    </>
  );
}

interface PushContentProps {
  state: PushState;
  errorMessage: string | null;
  onEnable: () => void;
  onDismiss: () => void;
}

function PushContent({ state, errorMessage, onEnable, onDismiss }: PushContentProps) {
  const positive = state === 'granted';
  const failure = state === 'denied' || state === 'unsupported' || state === 'error';

  const Icon = positive ? Bell : failure ? BellOff : Bell;
  const iconBg = positive ? T.okFill : failure ? T.subtle : T.amberFill;
  const iconInk = positive ? T.ok : failure ? T.ink3 : T.amberDp;

  const title =
    positive ? 'Уведомления включены' :
    state === 'denied'      ? 'Уведомления отключены' :
    state === 'unsupported' ? 'Уведомления недоступны' :
    state === 'error'       ? 'Не удалось включить' :
                              'Включить уведомления';

  const subtitle =
    positive ? 'Вы будете получать важные обновления'
             : 'Напоминания о задачах, изменениях в расписании и еженедельная сводка';

  return (
    <>
      <PromptHeader title={title} subtitle={subtitle} icon={Icon} iconBg={iconBg} iconInk={iconInk} />

      {state === 'denied' && (
        <Note>Включите уведомления для Personage в настройках браузера, затем перезагрузите страницу.</Note>
      )}
      {state === 'unsupported' && (
        <Note>Браузер не поддерживает push-уведомления. Можно продолжить без них.</Note>
      )}
      {state === 'error' && errorMessage && (
        <Note tone="danger">{errorMessage}</Note>
      )}

      <div style={{ display: 'flex', gap: 10 }}>
        <button type="button" onClick={onDismiss} style={secondaryBtn}>
          {positive ? 'Закрыть' : 'Не сейчас'}
        </button>
        {!positive && state !== 'denied' && state !== 'unsupported' && (
          <button
            type="button"
            onClick={onEnable}
            disabled={state === 'loading'}
            style={primaryBtn}
          >
            {state === 'loading'
              ? <Loader2 size={15} className="animate-spin" />
              : <Bell size={15} strokeWidth={2.2} />}
            Включить
          </button>
        )}
      </div>
    </>
  );
}

interface PromptHeaderProps {
  title: string;
  subtitle: string;
  icon?: LucideIcon;
  iconBg?: string;
  iconInk?: string;
}

function PromptHeader({ title, subtitle, icon: Icon, iconBg, iconInk }: PromptHeaderProps) {
  return (
    <div style={{
      display: 'flex', alignItems: 'center', gap: 12, marginBottom: 12,
    }}>
      {Icon ? (
        <div style={{
          width: 44, height: 44, borderRadius: 11,
          background: iconBg, color: iconInk,
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          flexShrink: 0,
        }}>
          <Icon size={22} strokeWidth={1.8} />
        </div>
      ) : (
        <div style={{
          width: 44, height: 44, borderRadius: 11,
          background: 'linear-gradient(135deg, oklch(0.45 0.07 55) 0%, oklch(0.32 0.04 55) 100%)',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          color: '#fff', fontFamily: SERIF, fontSize: 22, letterSpacing: -0.5,
          boxShadow: '0 3px 10px rgba(0,0,0,0.15), inset 0 1px 0 rgba(255,255,255,0.18)',
          flexShrink: 0,
        }}>P</div>
      )}
      <div style={{ minWidth: 0 }}>
        <div style={{
          fontFamily: SERIF, fontSize: 19, color: T.ink,
          letterSpacing: -0.2, lineHeight: 1.15,
        }}>{title}</div>
        <div style={{ fontSize: 12, color: T.ink3, marginTop: 1 }}>{subtitle}</div>
      </div>
    </div>
  );
}

interface PromptStepProps {
  n: number;
  title: ReactNode;
  visual: ReactNode;
  last?: boolean;
}

function PromptStep({ n, title, visual, last }: PromptStepProps) {
  return (
    <div style={{
      display: 'flex', alignItems: 'center', gap: 10,
      padding: '9px 12px',
      borderBottom: last ? 'none' : `0.5px solid ${T.hairline}`,
    }}>
      {visual}
      <div style={{ flex: 1, minWidth: 0, fontSize: 13, color: T.ink, lineHeight: 1.35 }}>
        <span style={{
          display: 'inline-block', width: 16, height: 16, borderRadius: '50%',
          background: T.subtleHi, color: T.ink2,
          fontSize: 10.5, fontWeight: 700,
          textAlign: 'center', lineHeight: '16px', marginRight: 6,
          verticalAlign: 1,
        }}>{n}</span>
        {title}
      </div>
    </div>
  );
}

function Note({ children, tone = 'info' }: { children: ReactNode; tone?: 'info' | 'danger' }) {
  return (
    <div style={{
      marginBottom: 12, padding: '10px 12px',
      borderRadius: 10,
      background: tone === 'danger' ? T.dangerFill : T.subtle,
      border: `0.5px solid ${T.hairline}`,
      fontSize: 12.5,
      color: tone === 'danger' ? T.danger : T.ink2,
      lineHeight: 1.45,
    }}>{children}</div>
  );
}

const secondaryBtn: React.CSSProperties = {
  flex: 1, padding: 12, borderRadius: 12,
  background: T.subtle, color: T.ink2,
  border: 'none', cursor: 'pointer',
  fontFamily: SANS, fontSize: 14, fontWeight: 500,
};

const primaryBtn: React.CSSProperties = {
  flex: 1.4, padding: 12, borderRadius: 12,
  background: T.ink, color: T.bg,
  border: 'none', cursor: 'pointer',
  fontFamily: SANS, fontSize: 14, fontWeight: 600,
  display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 6,
};

function visualBox(bg: string, ink: string): React.CSSProperties {
  return {
    width: 30, height: 30, borderRadius: 8,
    background: bg, color: ink,
    display: 'flex', alignItems: 'center', justifyContent: 'center',
  };
}


import { useEffect, useRef, useState, type ReactNode } from 'react';
import {
  ChevronLeft,
  Eye,
  EyeOff,
  KeyRound,
  Loader2,
  Lock,
  Mail,
  MailCheck,
  ShieldCheck,
  User,
  type LucideIcon,
} from 'lucide-react';
import { BrandMark } from '../mobile/Chrome';
import { SANS, SERIF, T } from '../mobile/tokens';
import { forgotPassword, login, register } from '../utils/authService';

const CONSENT_KEY = 'personage_consent_accepted';

type Mode = 'login' | 'register';
type View = 'auth' | 'forgot' | 'forgot-sent';

interface AuthScreenProps {
  onAuthSuccess: () => void;
}

const AuthScreen = ({ onAuthSuccess }: AuthScreenProps) => {
  const [view, setView] = useState<View>('auth');
  const [mode, setMode] = useState<Mode>('login');
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [showPwd, setShowPwd] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [consentRequired, setConsentRequired] = useState(false);
  const scrollerRef = useRef<HTMLDivElement>(null);

  // iOS auto-scrolls focused inputs above the autofill / Face ID popup,
  // pushing the screen up. Lock the scroll container while a credential
  // input is focused so the popup overlays the screen instead of moving it.
  useEffect(() => {
    const el = scrollerRef.current;
    if (!el) return;
    let lockedTop: number | null = null;
    const isCredentialInput = (t: EventTarget | null) =>
      t instanceof HTMLInputElement && (t.type === 'email' || t.type === 'password');
    const onFocusIn = (e: FocusEvent) => {
      if (isCredentialInput(e.target)) lockedTop = el.scrollTop;
    };
    const onFocusOut = () => { lockedTop = null; };
    const onScroll = () => {
      if (lockedTop !== null && el.scrollTop !== lockedTop) el.scrollTop = lockedTop;
    };
    el.addEventListener('focusin', onFocusIn);
    el.addEventListener('focusout', onFocusOut);
    el.addEventListener('scroll', onScroll, { passive: true });
    return () => {
      el.removeEventListener('focusin', onFocusIn);
      el.removeEventListener('focusout', onFocusOut);
      el.removeEventListener('scroll', onScroll);
    };
  }, []);

  const handleSubmit = () => {
    setError(null);
    if (!email || !password) {
      setError('Заполните email и пароль');
      return;
    }
    if (mode === 'register') {
      if (!name) { setError('Введите имя'); return; }
      if (password.length < 8) { setError('Пароль должен содержать не менее 8 символов'); return; }
      if (password !== confirmPassword) { setError('Пароли не совпадают'); return; }
    }
    if (localStorage.getItem(CONSENT_KEY) === 'true') {
      void runAuth();
    } else {
      setConsentRequired(true);
    }
  };

  const runAuth = async () => {
    setLoading(true);
    setError(null);
    try {
      if (mode === 'login') await login(email, password);
      else                  await register(email, password, name);
      onAuthSuccess();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Произошла ошибка');
    } finally {
      setLoading(false);
    }
  };

  const handleConsentAccepted = () => {
    localStorage.setItem(CONSENT_KEY, 'true');
    setConsentRequired(false);
    void runAuth();
  };

  if (view === 'forgot' || view === 'forgot-sent') {
    return (
      <ForgotView
        sent={view === 'forgot-sent'}
        defaultEmail={email}
        onBack={() => setView('auth')}
        onSent={(em) => { setEmail(em); setView('forgot-sent'); }}
      />
    );
  }

  return (
    <div ref={scrollerRef} style={{
      width: '100%', height: '100%',
      display: 'flex', flexDirection: 'column',
      background: T.bg, color: T.ink, fontFamily: SANS,
      paddingBottom: 34,
      overflowY: 'auto',
    }}>
      {/* Brand block */}
      <div style={{
        flex: 1, display: 'flex', flexDirection: 'column',
        alignItems: 'center', justifyContent: 'center',
        padding: '40px 28px 16px',
        textAlign: 'center',
      }}>
        <BrandMark size={84} />
        <div style={{
          fontFamily: SERIF, fontSize: 38, color: T.ink, letterSpacing: -0.6,
          marginTop: 24, lineHeight: 1.05,
        }}>Personage</div>
        <div style={{
          fontSize: 15, color: T.ink3, marginTop: 8,
          maxWidth: 280, lineHeight: 1.45,
        }}>
          Личный ассистент, который сам собирает задачи из почты, чатов и календаря
        </div>
      </div>

      <div style={{ padding: '0 20px 16px' }}>
        {/* Mode tabs */}
        <div style={{
          display: 'flex',
          background: T.subtle,
          borderRadius: 9,
          padding: 2,
          marginBottom: 16,
        }}>
          {(['login', 'register'] as Mode[]).map((m) => {
            const active = mode === m;
            return (
              <button
                type="button"
                key={m}
                onClick={() => { setMode(m); setError(null); }}
                style={{
                  flex: 1, padding: '9px 0', borderRadius: 7,
                  background: active ? T.surface : 'transparent',
                  border: 'none', cursor: 'pointer',
                  fontFamily: SANS, fontSize: 14, fontWeight: active ? 600 : 500,
                  color: active ? T.ink : T.ink2,
                  boxShadow: active ? '0 1px 2px rgba(0,0,0,0.06)' : 'none',
                }}
              >
                {m === 'login' ? 'Вход' : 'Регистрация'}
              </button>
            );
          })}
        </div>

        {error && (
          <div style={{
            marginBottom: 12, padding: '10px 12px',
            borderRadius: 10, background: T.dangerFill,
            border: `0.5px solid ${T.hairline}`,
            fontSize: 13, color: T.danger,
          }}>{error}</div>
        )}

        <div style={{
          background: T.surface, borderRadius: 14,
          border: `0.5px solid ${T.hairline}`,
          overflow: 'hidden',
          marginBottom: 14,
        }}>
          {mode === 'register' && (
            <Field icon={User} placeholder="Ваше имя" value={name} onChange={setName} />
          )}
          <Field
            icon={Mail}
            placeholder="Email"
            type="email"
            autoComplete="email"
            value={email}
            onChange={setEmail}
          />
          <Field
            icon={Lock}
            placeholder="Пароль"
            type={showPwd ? 'text' : 'password'}
            autoComplete={mode === 'register' ? 'new-password' : 'current-password'}
            value={password}
            onChange={setPassword}
            trailing={
              <button
                type="button"
                onClick={() => setShowPwd(!showPwd)}
                aria-label={showPwd ? 'Скрыть пароль' : 'Показать пароль'}
                style={{
                  background: 'transparent', border: 'none', cursor: 'pointer',
                  padding: 4, color: T.ink3,
                }}
              >
                {showPwd ? <EyeOff size={16} /> : <Eye size={16} />}
              </button>
            }
            last={mode !== 'register'}
          />
          {mode === 'register' && (
            <Field
              icon={Lock}
              placeholder="Повторите пароль"
              type={showPwd ? 'text' : 'password'}
              autoComplete="new-password"
              value={confirmPassword}
              onChange={setConfirmPassword}
              last
            />
          )}
        </div>

        {mode === 'login' && (
          <div style={{ textAlign: 'right', marginBottom: 14 }}>
            <button
              type="button"
              onClick={() => setView('forgot')}
              style={{
                background: 'transparent', border: 'none', cursor: 'pointer',
                fontFamily: SANS, fontSize: 13.5, color: T.amberDp, fontWeight: 500,
              }}
            >Забыли пароль?</button>
          </div>
        )}

        <button
          type="button"
          onClick={handleSubmit}
          disabled={loading}
          style={{
            width: '100%', padding: 15,
            background: T.ink, color: T.bg,
            border: 'none', borderRadius: 14,
            cursor: loading ? 'default' : 'pointer',
            fontFamily: SANS, fontSize: 16, fontWeight: 600,
            marginBottom: 14,
            display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 8,
            opacity: loading ? 0.7 : 1,
          }}
        >
          {loading && <Loader2 size={16} className="animate-spin" />}
          {mode === 'login' ? 'Войти' : 'Создать аккаунт'}
        </button>
      </div>

      <div style={{
        padding: '8px 28px 14px', textAlign: 'center',
        fontSize: 11.5, color: T.ink4, lineHeight: 1.5,
      }}>
        Продолжая, вы соглашаетесь с условиями использования и политикой конфиденциальности
      </div>

      {consentRequired && (
        <ConsentSheet
          onCancel={() => setConsentRequired(false)}
          onAgree={handleConsentAccepted}
        />
      )}
    </div>
  );
};

interface FieldProps {
  icon: LucideIcon;
  placeholder: string;
  type?: string;
  autoComplete?: string;
  value: string;
  onChange: (v: string) => void;
  trailing?: ReactNode;
  last?: boolean;
}

function Field({
  icon: Icon, placeholder, type = 'text', autoComplete,
  value, onChange, trailing, last,
}: FieldProps) {
  return (
    <div style={{
      display: 'flex', alignItems: 'center', gap: 10,
      padding: '13px 14px',
      borderBottom: last ? 'none' : `0.5px solid ${T.hairline}`,
    }}>
      <Icon size={16} strokeWidth={1.8} style={{ color: T.ink3, flexShrink: 0 }} />
      <input
        value={value}
        onChange={(e) => onChange(e.target.value)}
        type={type}
        autoComplete={autoComplete}
        placeholder={placeholder}
        style={{
          flex: 1, background: 'transparent', border: 'none', outline: 'none',
          fontFamily: SANS, fontSize: 16, color: T.ink, minWidth: 0,
        }}
      />
      {trailing}
    </div>
  );
}

interface ForgotViewProps {
  sent: boolean;
  defaultEmail: string;
  onBack: () => void;
  onSent: (email: string) => void;
}

function ForgotView({ sent, defaultEmail, onBack, onSent }: ForgotViewProps) {
  const [email, setEmail] = useState(defaultEmail);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async () => {
    setError(null);
    if (!email) { setError('Введите email'); return; }
    setLoading(true);
    try {
      await forgotPassword(email, window.location.origin);
      onSent(email);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось отправить письмо');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{
      width: '100%', height: '100%',
      display: 'flex', flexDirection: 'column',
      background: T.bg, color: T.ink, fontFamily: SANS,
      paddingTop: 50, paddingBottom: 34,
    }}>
      <div style={{ padding: '0 12px 4px' }}>
        <button
          type="button"
          onClick={onBack}
          style={{
            background: 'transparent', border: 'none', cursor: 'pointer',
            padding: '8px 10px', color: T.amberDp,
            fontFamily: SANS, fontSize: 16, fontWeight: 400,
            display: 'inline-flex', alignItems: 'center', gap: 2,
          }}
        >
          <ChevronLeft size={20} strokeWidth={2.2} />
          Назад
        </button>
      </div>

      <div style={{
        flex: 1, padding: '8px 24px 16px',
        display: 'flex', flexDirection: 'column',
      }}>
        <div style={{
          width: 56, height: 56, borderRadius: '50%', background: T.amberFill,
          color: T.amberDp,
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          marginBottom: 18,
        }}>
          {sent
            ? <MailCheck size={26} strokeWidth={1.7} />
            : <KeyRound  size={26} strokeWidth={1.7} />}
        </div>

        <div style={{
          fontFamily: SERIF, fontSize: 30, color: T.ink, letterSpacing: -0.4, lineHeight: 1.1,
          marginBottom: 10,
        }}>
          {sent ? 'Письмо отправлено' : 'Восстановление пароля'}
        </div>
        <div style={{
          fontSize: 14.5, color: T.ink3, lineHeight: 1.5, marginBottom: 22,
        }}>
          {sent
            ? 'Мы отправили ссылку для сброса пароля на ваш email. Проверьте папку «Входящие» и «Спам».'
            : 'Введите email от вашего аккаунта — мы пришлём ссылку для сброса пароля.'}
        </div>

        {error && (
          <div style={{
            marginBottom: 12, padding: '10px 12px',
            borderRadius: 10, background: T.dangerFill,
            border: `0.5px solid ${T.hairline}`,
            fontSize: 13, color: T.danger,
          }}>{error}</div>
        )}

        {!sent && (
          <div style={{
            background: T.surface, borderRadius: 14,
            border: `0.5px solid ${T.hairline}`,
            overflow: 'hidden', marginBottom: 16,
          }}>
            <Field
              icon={Mail}
              placeholder="Email"
              type="email"
              autoComplete="email"
              value={email}
              onChange={setEmail}
              last
            />
          </div>
        )}

        {sent ? (
          <button
            type="button"
            onClick={onBack}
            style={primaryButton(false)}
          >Вернуться к входу</button>
        ) : (
          <button
            type="button"
            onClick={() => void handleSubmit()}
            disabled={loading}
            style={primaryButton(loading)}
          >
            {loading && <Loader2 size={16} className="animate-spin" />}
            Отправить ссылку
          </button>
        )}

        {!sent && (
          <div style={{
            marginTop: 14, textAlign: 'center',
            fontSize: 13.5, color: T.ink3,
          }}>
            Вспомнили пароль?{' '}
            <button
              type="button"
              onClick={onBack}
              style={{
                background: 'transparent', border: 'none', cursor: 'pointer',
                fontFamily: SANS, fontSize: 13.5, color: T.amberDp, fontWeight: 600, padding: 0,
              }}
            >Войти</button>
          </div>
        )}
      </div>
    </div>
  );
}

function primaryButton(loading: boolean): React.CSSProperties {
  return {
    width: '100%', padding: 15,
    background: T.ink, color: T.bg,
    border: 'none', borderRadius: 14,
    cursor: loading ? 'default' : 'pointer',
    fontFamily: SANS, fontSize: 16, fontWeight: 600,
    display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 8,
    opacity: loading ? 0.7 : 1,
  };
}

interface ConsentSheetProps {
  onCancel: () => void;
  onAgree: () => void;
}

function ConsentSheet({ onCancel, onAgree }: ConsentSheetProps) {
  const [agreedPolicy, setAgreedPolicy] = useState(false);
  const [agreedData, setAgreedData] = useState(false);
  const can = agreedPolicy && agreedData;

  return (
    <div
      className="animate-consent-fade"
      style={{
        position: 'absolute', inset: 0, zIndex: 200,
        background: 'rgba(20,16,8,0.45)',
        backdropFilter: 'blur(2px)',
        display: 'flex', alignItems: 'flex-end', justifyContent: 'center',
      }}
    >
      <div
        className="animate-consent-rise"
        style={{
          width: '100%', background: T.bg, color: T.ink,
          borderRadius: '20px 20px 0 0',
          padding: '14px 20px 28px',
          boxShadow: '0 -10px 32px rgba(0,0,0,0.18)',
          maxHeight: '88%',
          display: 'flex', flexDirection: 'column',
          fontFamily: SANS,
        }}
      >
        <div style={{
          width: 36, height: 5, borderRadius: 99,
          background: T.subtleHi, margin: '0 auto 14px',
        }} />

        <div style={{
          width: 48, height: 48, borderRadius: 14, background: T.amberFill,
          color: T.amberDp,
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          marginBottom: 12,
        }}>
          <ShieldCheck size={24} strokeWidth={1.8} />
        </div>

        <div style={{
          fontFamily: SERIF, fontSize: 26, color: T.ink,
          letterSpacing: -0.3, lineHeight: 1.12,
          marginBottom: 8,
        }}>Обработка персональных данных</div>
        <div style={{
          fontSize: 13.5, color: T.ink3, lineHeight: 1.5, marginBottom: 14,
        }}>
          Чтобы продолжить, ознакомьтесь с условиями и подтвердите согласие. Без этого мы не сможем создать аккаунт.
        </div>

        <div style={{
          flex: 1, minHeight: 0, overflow: 'auto',
          background: T.surface, borderRadius: 12,
          border: `0.5px solid ${T.hairline}`,
          padding: '12px 14px', marginBottom: 14,
          fontSize: 12.5, color: T.ink2, lineHeight: 1.55,
        }}>
          <div style={{ fontWeight: 600, color: T.ink, marginBottom: 6 }}>
            Какие данные мы обрабатываем
          </div>
          <div style={{ marginBottom: 10 }}>
            Email, имя, содержимое подключённых источников (Gmail, Telegram) — только в объёме, необходимом для извлечения задач. Данные хранятся в зашифрованном виде на серверах в РФ.
          </div>
          <div style={{ fontWeight: 600, color: T.ink, marginBottom: 6 }}>
            Как используем
          </div>
          <div style={{ marginBottom: 10 }}>
            Только для работы сервиса: распознавание задач, напоминания, синхронизация. Не передаём третьим лицам, не используем для рекламы.
          </div>
          <div style={{ fontWeight: 600, color: T.ink, marginBottom: 6 }}>
            Ваши права
          </div>
          <div>
            Вы можете в любой момент отозвать согласие, удалить аккаунт и все данные через настройки.
          </div>
        </div>

        <ConsentCheck
          checked={agreedPolicy}
          onToggle={() => setAgreedPolicy(!agreedPolicy)}
          label={
            <>
              Я прочитал(а) и принимаю{' '}
              <span style={{ color: T.amberDp, fontWeight: 500 }}>Политику конфиденциальности</span>
              {' '}и{' '}
              <span style={{ color: T.amberDp, fontWeight: 500 }}>Условия использования</span>
            </>
          }
        />
        <ConsentCheck
          checked={agreedData}
          onToggle={() => setAgreedData(!agreedData)}
          label={<>Согласен(на) на обработку персональных данных в соответствии с 152-ФЗ</>}
        />

        <div style={{ display: 'flex', gap: 10, marginTop: 16 }}>
          <button
            type="button"
            onClick={onCancel}
            style={{
              flex: 1, padding: 14, borderRadius: 12,
              background: T.subtle, color: T.ink2,
              border: 'none', cursor: 'pointer',
              fontFamily: SANS, fontSize: 15, fontWeight: 500,
            }}
          >Отмена</button>
          <button
            type="button"
            onClick={can ? onAgree : undefined}
            disabled={!can}
            style={{
              flex: 1.4, padding: 14, borderRadius: 12,
              background: can ? T.ink : T.subtleHi,
              color: can ? T.bg : T.ink4,
              border: 'none', cursor: can ? 'pointer' : 'not-allowed',
              fontFamily: SANS, fontSize: 15, fontWeight: 600,
            }}
          >Принять и продолжить</button>
        </div>
      </div>
    </div>
  );
}

interface ConsentCheckProps {
  checked: boolean;
  onToggle: () => void;
  label: ReactNode;
}

function ConsentCheck({ checked, onToggle, label }: ConsentCheckProps) {
  return (
    <button
      type="button"
      onClick={onToggle}
      style={{
        display: 'flex', gap: 10, alignItems: 'flex-start',
        background: 'transparent', border: 'none', cursor: 'pointer',
        padding: '8px 2px', textAlign: 'left',
        fontFamily: SANS,
      }}
    >
      <div style={{
        width: 20, height: 20, borderRadius: 6,
        border: `1.5px solid ${checked ? T.amberDp : T.hairline}`,
        background: checked ? T.amberDp : 'transparent',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        flexShrink: 0, marginTop: 1,
        transition: 'all .15s',
      }}>
        {checked && (
          <svg width="11" height="9" viewBox="0 0 11 9">
            <path d="M1 4.5L4 7.5L10 1.5" stroke="#fff" strokeWidth="2"
              strokeLinecap="round" strokeLinejoin="round" fill="none" />
          </svg>
        )}
      </div>
      <div style={{ fontSize: 13, color: T.ink2, lineHeight: 1.45 }}>{label}</div>
    </button>
  );
}

export default AuthScreen;

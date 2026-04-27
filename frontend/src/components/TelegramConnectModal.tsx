import { useEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import QRCode from 'qrcode';
import { ChevronDown, Loader2, RefreshCw } from 'lucide-react';
import { SANS, SERIF, T } from '../mobile/tokens';
import {
  getTelegramAuthStatus,
  initiateTelegramAuth,
  resendTelegramCode,
  verifyTelegramCode,
  type TelegramAuthMethod,
} from '../utils/telegramAuthService';

interface TelegramConnectModalProps {
  userId: string;
  onClose: () => void;
  onSuccess: () => void;
}

type Step = 'qr' | 'phone-input' | 'code-input' | 'password-input';

const QR_POLL_INTERVAL_MS = 2000;

export function TelegramConnectModal({ userId, onClose, onSuccess }: TelegramConnectModalProps) {
  const [method, setMethod] = useState<TelegramAuthMethod>('qr');
  const [step, setStep] = useState<Step>('qr');
  const [loginId, setLoginId] = useState<string | null>(null);
  const [qrSvg, setQrSvg] = useState<string | null>(null);
  const [qrExpired, setQrExpired] = useState(false);
  const [phone, setPhone] = useState('');
  const [code, setCode] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [resentNote, setResentNote] = useState<string | null>(null);
  const [mountTarget, setMountTarget] = useState<HTMLElement | null>(null);

  const onSuccessRef = useRef(onSuccess);
  useEffect(() => { onSuccessRef.current = onSuccess; }, [onSuccess]);

  useEffect(() => {
    setMountTarget(document.getElementById('mobile-frame') ?? document.body);
  }, []);

  useEffect(() => {
    if (method !== 'qr') return;
    let cancelled = false;
    setStep('qr');
    setError(null);
    setQrExpired(false);
    setQrSvg(null);
    setLoginId(null);
    setLoading(true);

    void (async () => {
      try {
        const res = await initiateTelegramAuth(userId, 'qr');
        if (cancelled) return;
        setLoginId(res.login_id);
        if (res.qr_data) {
          const svg = await QRCode.toString(res.qr_data, {
            type: 'svg',
            margin: 1,
            color: { dark: '#1a1a1a', light: '#00000000' },
            errorCorrectionLevel: 'M',
          });
          if (!cancelled) setQrSvg(svg);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Не удалось получить QR-код');
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();

    return () => { cancelled = true; };
  }, [method, userId]);

  useEffect(() => {
    if (method !== 'qr' || !loginId || qrExpired) return;
    let stopped = false;
    const poll = async () => {
      if (stopped) return;
      try {
        const res = await getTelegramAuthStatus(loginId);
        if (stopped) return;
        if (res.status === 'completed') {
          stopped = true;
          onSuccessRef.current();
          return;
        }
        if (res.status === 'expired' || res.status === 'failed') {
          stopped = true;
          setQrExpired(true);
          return;
        }
      } catch {
        // ignore transient polling errors
      }
    };
    const id = window.setInterval(() => { void poll(); }, QR_POLL_INTERVAL_MS);
    return () => { stopped = true; window.clearInterval(id); };
  }, [method, loginId, qrExpired]);

  const handleRefreshQr = () => {
    setMethod('qr');
    setQrExpired(false);
    setStep('qr');
    setLoginId(null);
    setQrSvg(null);
    setLoading(true);
    void (async () => {
      try {
        const res = await initiateTelegramAuth(userId, 'qr');
        setLoginId(res.login_id);
        if (res.qr_data) {
          const svg = await QRCode.toString(res.qr_data, {
            type: 'svg',
            margin: 1,
            color: { dark: '#1a1a1a', light: '#00000000' },
            errorCorrectionLevel: 'M',
          });
          setQrSvg(svg);
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Не удалось обновить QR-код');
      } finally {
        setLoading(false);
      }
    })();
  };

  const handlePhoneSubmit = async () => {
    if (!phone.trim()) return;
    setError(null);
    setSubmitting(true);
    try {
      const res = await initiateTelegramAuth(userId, 'phone', phone.trim());
      setLoginId(res.login_id);
      setStep('code-input');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось отправить код');
    } finally {
      setSubmitting(false);
    }
  };

  const handleCodeSubmit = async () => {
    if (!loginId || !code.trim()) return;
    setError(null);
    setSubmitting(true);
    try {
      const res = await verifyTelegramCode(loginId, code.trim());
      if (res.status === 'success') {
        onSuccess();
      } else if (res.status === 'password_required') {
        setStep('password-input');
      } else {
        setError(res.message ?? 'Неверный код');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка проверки кода');
    } finally {
      setSubmitting(false);
    }
  };

  const handlePasswordSubmit = async () => {
    if (!loginId || !password) return;
    setError(null);
    setSubmitting(true);
    try {
      const res = await verifyTelegramCode(loginId, code.trim(), password);
      if (res.status === 'success') {
        onSuccess();
      } else {
        setError(res.message ?? 'Неверный пароль');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка проверки пароля');
    } finally {
      setSubmitting(false);
    }
  };

  const handleResend = async () => {
    if (!loginId) return;
    setError(null);
    setResentNote(null);
    try {
      await resendTelegramCode(loginId);
      setResentNote('Код отправлен повторно');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось отправить код повторно');
    }
  };

  if (!mountTarget) return null;

  const sheet = (
    <div
      className="animate-sheet-in"
      style={{
        position: 'absolute', inset: 0, zIndex: 100,
        display: 'flex', flexDirection: 'column',
        background: T.bg, color: T.ink, fontFamily: SANS,
        overflow: 'hidden',
      }}
    >
      <div style={{
        flexShrink: 0, padding: '8px 0 4px',
        display: 'flex', justifyContent: 'center',
        background: T.bg,
      }}>
        <div style={{ width: 36, height: 5, borderRadius: 99, background: T.subtleHi }} />
      </div>

      <div style={{
        flexShrink: 0,
        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        padding: '6px 12px 10px',
      }}>
        <button
          type="button"
          onClick={onClose}
          style={{
            background: 'transparent', border: 'none', cursor: 'pointer',
            padding: '6px 10px', color: T.amberDp,
            fontFamily: SANS, fontSize: 16, fontWeight: 400,
            display: 'flex', alignItems: 'center', gap: 2,
          }}
        >
          <ChevronDown size={20} strokeWidth={2.2} />
          Закрыть
        </button>
        <div style={{ width: 40 }} />
      </div>

      <div style={{ flex: 1, minHeight: 0, overflow: 'auto', padding: '0 18px 24px' }}>
        <div style={{
          fontFamily: SERIF, fontSize: 28, lineHeight: 1.15, letterSpacing: -0.4,
          color: T.ink, marginBottom: 8,
        }}>
          Подключение Telegram
        </div>
        <div style={{
          fontSize: 14, color: T.ink2, lineHeight: 1.45, marginBottom: 18,
        }}>
          Подключите свой аккаунт Telegram, чтобы получать задачи из переписок.
        </div>

        {/* Method tabs */}
        <div style={{
          display: 'flex', gap: 6, padding: 4,
          background: T.surface, borderRadius: 12,
          border: `0.5px solid ${T.hairline}`,
          marginBottom: 18,
        }}>
          <MethodTab
            label="QR-код"
            active={method === 'qr'}
            onClick={() => {
              setMethod('qr');
              setError(null);
              setResentNote(null);
            }}
          />
          <MethodTab
            label="По телефону"
            active={method === 'phone'}
            onClick={() => {
              setMethod('phone');
              setStep('phone-input');
              setError(null);
              setResentNote(null);
              setLoginId(null);
              setPhone('');
              setCode('');
              setPassword('');
            }}
          />
        </div>

        {error && (
          <div style={{
            padding: '10px 12px', marginBottom: 14,
            fontSize: 13, color: T.danger, background: T.dangerFill,
            borderRadius: 10, lineHeight: 1.4,
          }}>{error}</div>
        )}

        {method === 'qr' && (
          <QrSection
            loading={loading}
            qrSvg={qrSvg}
            expired={qrExpired}
            onRefresh={handleRefreshQr}
          />
        )}

        {method === 'phone' && step === 'phone-input' && (
          <PhoneStep
            phone={phone}
            onPhoneChange={setPhone}
            submitting={submitting}
            onSubmit={() => void handlePhoneSubmit()}
          />
        )}

        {method === 'phone' && step === 'code-input' && (
          <CodeStep
            phone={phone}
            code={code}
            onCodeChange={setCode}
            submitting={submitting}
            onSubmit={() => void handleCodeSubmit()}
            onResend={() => void handleResend()}
            resentNote={resentNote}
          />
        )}

        {method === 'phone' && step === 'password-input' && (
          <PasswordStep
            password={password}
            onPasswordChange={setPassword}
            submitting={submitting}
            onSubmit={() => void handlePasswordSubmit()}
          />
        )}
      </div>
    </div>
  );

  return createPortal(sheet, mountTarget);
}

function MethodTab({ label, active, onClick }: { label: string; active: boolean; onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      style={{
        flex: 1, padding: '8px 12px', borderRadius: 8,
        background: active ? T.amberDp : 'transparent',
        color: active ? '#fff' : T.ink2,
        border: 'none', cursor: 'pointer',
        fontFamily: SANS, fontSize: 13.5, fontWeight: 600,
        transition: 'background .15s, color .15s',
      }}
    >{label}</button>
  );
}

function QrSection({
  loading, qrSvg, expired, onRefresh,
}: { loading: boolean; qrSvg: string | null; expired: boolean; onRefresh: () => void }) {
  return (
    <div style={{
      background: T.surface, border: `0.5px solid ${T.hairline}`,
      borderRadius: 14, padding: 18,
      display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 14,
    }}>
      <div style={{
        width: 220, height: 220, position: 'relative',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
      }}>
        {loading && <Loader2 size={28} className="animate-spin" style={{ color: T.ink3 }} />}
        {!loading && qrSvg && (
          <div
            style={{ width: '100%', height: '100%', opacity: expired ? 0.25 : 1 }}
            dangerouslySetInnerHTML={{ __html: qrSvg }}
          />
        )}
        {expired && (
          <button
            type="button"
            onClick={onRefresh}
            style={{
              position: 'absolute', inset: 0, margin: 'auto',
              width: 'max-content', height: 'max-content',
              padding: '10px 16px', borderRadius: 999,
              background: T.amberDp, color: '#fff', border: 'none',
              cursor: 'pointer', fontFamily: SANS, fontSize: 13.5, fontWeight: 600,
              display: 'inline-flex', alignItems: 'center', gap: 6,
            }}
          >
            <RefreshCw size={14} strokeWidth={2} /> Обновить QR
          </button>
        )}
      </div>
      <div style={{
        fontSize: 13.5, color: T.ink2, textAlign: 'center', lineHeight: 1.5, maxWidth: 280,
      }}>
        Откройте Telegram → Настройки → Устройства → Подключить устройство и отсканируйте код.
      </div>
    </div>
  );
}

function PhoneStep({
  phone, onPhoneChange, submitting, onSubmit,
}: { phone: string; onPhoneChange: (v: string) => void; submitting: boolean; onSubmit: () => void }) {
  return (
    <form
      onSubmit={(e) => { e.preventDefault(); onSubmit(); }}
      style={{
        display: 'flex', flexDirection: 'column', gap: 12,
      }}
    >
      <Label>Номер телефона</Label>
      <Input
        type="tel"
        value={phone}
        onChange={(e) => onPhoneChange(e.target.value)}
        placeholder="+7 999 123 45 67"
        autoFocus
      />
      <div style={{ fontSize: 12.5, color: T.ink3, lineHeight: 1.4 }}>
        Укажите номер в международном формате. Telegram отправит код в приложение.
      </div>
      <PrimaryButton type="submit" disabled={submitting || !phone.trim()} loading={submitting}>
        Отправить код
      </PrimaryButton>
    </form>
  );
}

function CodeStep({
  phone, code, onCodeChange, submitting, onSubmit, onResend, resentNote,
}: {
  phone: string;
  code: string;
  onCodeChange: (v: string) => void;
  submitting: boolean;
  onSubmit: () => void;
  onResend: () => void;
  resentNote: string | null;
}) {
  return (
    <form
      onSubmit={(e) => { e.preventDefault(); onSubmit(); }}
      style={{ display: 'flex', flexDirection: 'column', gap: 12 }}
    >
      <Label>Код подтверждения</Label>
      <div style={{ fontSize: 12.5, color: T.ink3, lineHeight: 1.4 }}>
        Код отправлен в Telegram на {phone}.
      </div>
      <Input
        type="text"
        inputMode="numeric"
        value={code}
        onChange={(e) => onCodeChange(e.target.value)}
        placeholder="12345"
        autoFocus
      />
      {resentNote && (
        <div style={{ fontSize: 12.5, color: T.ok }}>{resentNote}</div>
      )}
      <PrimaryButton type="submit" disabled={submitting || !code.trim()} loading={submitting}>
        Подтвердить
      </PrimaryButton>
      <button
        type="button"
        onClick={onResend}
        style={{
          background: 'transparent', border: 'none',
          color: T.amberDp, cursor: 'pointer',
          fontFamily: SANS, fontSize: 13.5, fontWeight: 500,
          padding: '4px 0',
        }}
      >
        Отправить код повторно
      </button>
    </form>
  );
}

function PasswordStep({
  password, onPasswordChange, submitting, onSubmit,
}: { password: string; onPasswordChange: (v: string) => void; submitting: boolean; onSubmit: () => void }) {
  return (
    <form
      onSubmit={(e) => { e.preventDefault(); onSubmit(); }}
      style={{ display: 'flex', flexDirection: 'column', gap: 12 }}
    >
      <Label>Облачный пароль</Label>
      <div style={{ fontSize: 12.5, color: T.ink3, lineHeight: 1.4 }}>
        Включена двухфакторная аутентификация. Введите облачный пароль Telegram.
      </div>
      <Input
        type="password"
        value={password}
        onChange={(e) => onPasswordChange(e.target.value)}
        placeholder="Облачный пароль"
        autoFocus
      />
      <PrimaryButton type="submit" disabled={submitting || !password} loading={submitting}>
        Войти
      </PrimaryButton>
    </form>
  );
}

function Label({ children }: { children: React.ReactNode }) {
  return (
    <label style={{
      fontSize: 12, color: T.ink3, fontWeight: 600,
      letterSpacing: 0.3, textTransform: 'uppercase',
    }}>{children}</label>
  );
}

function Input(props: React.InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      {...props}
      style={{
        padding: '12px 14px', borderRadius: 10,
        border: `0.5px solid ${T.hairline}`,
        background: T.surface, color: T.ink,
        fontFamily: SANS, fontSize: 16,
        outline: 'none',
        ...props.style,
      }}
    />
  );
}

function PrimaryButton({
  type = 'button',
  disabled,
  loading,
  onClick,
  children,
}: {
  type?: 'button' | 'submit';
  disabled?: boolean;
  loading?: boolean;
  onClick?: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type={type}
      onClick={onClick}
      disabled={disabled}
      style={{
        marginTop: 4,
        padding: '12px 16px', borderRadius: 12,
        background: T.ink, color: T.bg, border: 'none',
        cursor: disabled ? 'default' : 'pointer',
        fontFamily: SANS, fontSize: 15, fontWeight: 600,
        opacity: disabled ? 0.5 : 1,
        display: 'inline-flex', alignItems: 'center', justifyContent: 'center', gap: 6,
      }}
    >
      {loading && <Loader2 size={15} className="animate-spin" />}
      {children}
    </button>
  );
}

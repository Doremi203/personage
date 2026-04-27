import { useState } from 'react';
import { CheckCircle2, Eye, EyeOff, KeyRound, Loader2, Lock } from 'lucide-react';
import { SANS, SERIF, T } from '../mobile/tokens';
import { resetPassword } from '../utils/authService';

interface ResetPasswordScreenProps {
  token: string;
  onSuccess: () => void;
}

const ResetPasswordScreen = ({ token, onSuccess }: ResetPasswordScreenProps) => {
  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [showPwd, setShowPwd] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  const handleSubmit = async () => {
    setError(null);
    if (password !== confirm) { setError('Пароли не совпадают'); return; }
    if (password.length < 8)  { setError('Пароль должен содержать не менее 8 символов'); return; }
    setLoading(true);
    try {
      await resetPassword(token, password);
      setSuccess(true);
      setTimeout(onSuccess, 1500);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось сбросить пароль');
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
          {success
            ? <CheckCircle2 size={26} strokeWidth={1.7} style={{ color: T.ok }} />
            : <KeyRound     size={26} strokeWidth={1.7} />}
        </div>

        <div style={{
          fontFamily: SERIF, fontSize: 30, color: T.ink, letterSpacing: -0.4, lineHeight: 1.1,
          marginBottom: 10,
        }}>
          {success ? 'Пароль изменён' : 'Новый пароль'}
        </div>
        <div style={{ fontSize: 14.5, color: T.ink3, lineHeight: 1.5, marginBottom: 22 }}>
          {success
            ? 'Выполняется вход…'
            : 'Придумайте надёжный пароль (не менее 8 символов).'}
        </div>

        {error && (
          <div style={{
            marginBottom: 12, padding: '10px 12px',
            borderRadius: 10, background: T.dangerFill,
            border: `0.5px solid ${T.hairline}`,
            fontSize: 13, color: T.danger,
          }}>{error}</div>
        )}

        {!success && (
          <>
            <div style={{
              background: T.surface, borderRadius: 14,
              border: `0.5px solid ${T.hairline}`,
              overflow: 'hidden', marginBottom: 16,
            }}>
              <div style={{
                display: 'flex', alignItems: 'center', gap: 10,
                padding: '13px 14px',
                borderBottom: `0.5px solid ${T.hairline}`,
              }}>
                <Lock size={16} strokeWidth={1.8} style={{ color: T.ink3, flexShrink: 0 }} />
                <input
                  type={showPwd ? 'text' : 'password'}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  autoComplete="new-password"
                  placeholder="Новый пароль"
                  style={{
                    flex: 1, background: 'transparent', border: 'none', outline: 'none',
                    fontFamily: SANS, fontSize: 16, color: T.ink, minWidth: 0,
                  }}
                />
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
              </div>
              <div style={{
                display: 'flex', alignItems: 'center', gap: 10,
                padding: '13px 14px',
              }}>
                <Lock size={16} strokeWidth={1.8} style={{ color: T.ink3, flexShrink: 0 }} />
                <input
                  type={showPwd ? 'text' : 'password'}
                  value={confirm}
                  onChange={(e) => setConfirm(e.target.value)}
                  autoComplete="new-password"
                  placeholder="Подтверждение пароля"
                  style={{
                    flex: 1, background: 'transparent', border: 'none', outline: 'none',
                    fontFamily: SANS, fontSize: 16, color: T.ink, minWidth: 0,
                  }}
                />
              </div>
            </div>

            <button
              type="button"
              onClick={() => void handleSubmit()}
              disabled={loading}
              style={{
                width: '100%', padding: 15,
                background: T.ink, color: T.bg,
                border: 'none', borderRadius: 14,
                cursor: loading ? 'default' : 'pointer',
                fontFamily: SANS, fontSize: 16, fontWeight: 600,
                display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 8,
                opacity: loading ? 0.7 : 1,
              }}
            >
              {loading && <Loader2 size={16} className="animate-spin" />}
              Сохранить пароль
            </button>
          </>
        )}
      </div>
    </div>
  );
};

export default ResetPasswordScreen;

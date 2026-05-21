import { useEffect, useMemo, useState } from 'react';
import { createPortal } from 'react-dom';
import { ChevronDown, Loader2, RefreshCw, Search } from 'lucide-react';
import { SANS, SERIF, T } from '../mobile/tokens';
import {
  getUserChats,
  updateUserChat,
  type TelegramChat,
} from '../utils/telegramChatsService';

interface TelegramChatsSheetProps {
  onClose: () => void;
}

interface ChatRow extends TelegramChat {
  toggling: boolean;
}

export function TelegramChatsSheet({ onClose }: TelegramChatsSheetProps) {
  const [chats, setChats] = useState<ChatRow[] | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [toggleError, setToggleError] = useState<string | null>(null);
  const [query, setQuery] = useState('');
  const [mountTarget, setMountTarget] = useState<HTMLElement | null>(null);

  useEffect(() => {
    setMountTarget(document.getElementById('mobile-frame') ?? document.body);
  }, []);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      setError(null);
      setLoading(true);
      try {
        const data = await getUserChats();
        if (cancelled) return;
        setChats(data.map((c) => ({ ...c, toggling: false })));
      } catch (err) {
        if (cancelled) return;
        setError(err instanceof Error ? err.message : 'Не удалось загрузить чаты');
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => { cancelled = true; };
  }, []);

  const handleRefresh = async () => {
    if (refreshing || loading) return;
    setRefreshing(true);
    setError(null);
    setToggleError(null);
    try {
      const data = await getUserChats();
      setChats(data.map((c) => ({ ...c, toggling: false })));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось обновить список чатов');
    } finally {
      setRefreshing(false);
    }
  };

  const handleToggle = (chatId: number, nextValue: boolean) => {
    setToggleError(null);
    setChats((prev) =>
      prev ? prev.map((c) =>
        c.chatId === chatId ? { ...c, isActive: nextValue, toggling: true } : c,
      ) : prev,
    );
    void updateUserChat(chatId, nextValue)
      .then(() => {
        setChats((prev) =>
          prev ? prev.map((c) =>
            c.chatId === chatId ? { ...c, toggling: false } : c,
          ) : prev,
        );
      })
      .catch((err: unknown) => {
        setChats((prev) =>
          prev ? prev.map((c) =>
            c.chatId === chatId ? { ...c, isActive: !nextValue, toggling: false } : c,
          ) : prev,
        );
        setToggleError(err instanceof Error ? err.message : 'Не удалось изменить статус чата');
      });
  };

  const filteredChats = useMemo(() => {
    if (!chats) return null;
    const trimmed = query.trim().toLowerCase();
    if (!trimmed) return chats;
    return chats.filter((c) => c.chatName.toLowerCase().includes(trimmed));
  }, [chats, query]);

  const activeCount = chats?.filter((c) => c.isActive).length ?? 0;

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
        <button
          type="button"
          onClick={() => void handleRefresh()}
          disabled={refreshing || loading}
          aria-label="Обновить список чатов"
          style={{
            background: 'transparent', border: 'none',
            cursor: refreshing || loading ? 'default' : 'pointer',
            padding: '6px 10px', color: T.amberDp,
            display: 'inline-flex', alignItems: 'center', gap: 6,
            fontFamily: SANS, fontSize: 14,
            opacity: refreshing || loading ? 0.5 : 1,
          }}
        >
          {refreshing
            ? <Loader2 size={16} className="animate-spin" />
            : <RefreshCw size={16} strokeWidth={2} />}
          Обновить
        </button>
      </div>

      <div style={{ flex: 1, minHeight: 0, overflow: 'auto', padding: '0 18px 24px' }}>
        <div style={{
          fontFamily: SERIF, fontSize: 28, lineHeight: 1.15, letterSpacing: -0.4,
          color: T.ink, marginBottom: 6,
        }}>
          Чаты Telegram
        </div>
        <div style={{
          fontSize: 14, color: T.ink2, lineHeight: 1.45, marginBottom: 14,
        }}>
          {loading
            ? 'Загружаем список чатов из Telegram. Это может занять несколько секунд.'
            : chats && chats.length > 0
              ? `Выберите чаты, из которых получать задачи. Активно: ${activeCount} из ${chats.length}.`
              : 'Подключите чаты, чтобы получать из них задачи.'}
        </div>

        {error && (
          <div style={{
            padding: '10px 12px', marginBottom: 14,
            fontSize: 13, color: T.danger, background: T.dangerFill,
            borderRadius: 10, lineHeight: 1.4,
          }}>{error}</div>
        )}

        {toggleError && (
          <div style={{
            padding: '10px 12px', marginBottom: 14,
            fontSize: 13, color: T.danger, background: T.dangerFill,
            borderRadius: 10, lineHeight: 1.4,
          }}>{toggleError}</div>
        )}

        {!loading && chats && chats.length > 0 && (
          <div style={{
            display: 'flex', alignItems: 'center', gap: 8,
            background: T.surface, border: `0.5px solid ${T.hairline}`,
            borderRadius: 10, padding: '8px 12px', marginBottom: 12,
          }}>
            <Search size={15} strokeWidth={1.8} style={{ color: T.ink3, flexShrink: 0 }} />
            <input
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Поиск по названию"
              style={{
                flex: 1, border: 'none', outline: 'none',
                background: 'transparent', color: T.ink,
                fontFamily: SANS, fontSize: 14,
              }}
            />
          </div>
        )}

        {loading && (
          <div style={{
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            padding: '40px 0', color: T.ink3,
          }}>
            <Loader2 size={22} className="animate-spin" />
          </div>
        )}

        {!loading && chats && chats.length === 0 && (
          <div style={{
            padding: '24px 14px', textAlign: 'center',
            color: T.ink3, fontSize: 14, lineHeight: 1.5,
          }}>
            Не нашли ни одного чата. Убедитесь, что в Telegram есть переписки, и нажмите «Обновить».
          </div>
        )}

        {!loading && filteredChats && filteredChats.length === 0 && chats && chats.length > 0 && (
          <div style={{
            padding: '24px 14px', textAlign: 'center',
            color: T.ink3, fontSize: 14, lineHeight: 1.5,
          }}>
            Ничего не нашли по запросу «{query.trim()}».
          </div>
        )}

        {!loading && filteredChats && filteredChats.length > 0 && (
          <div style={{
            background: T.surface, borderRadius: 14,
            border: `0.5px solid ${T.hairline}`,
            overflow: 'hidden',
          }}>
            {filteredChats.map((c, i) => (
              <ChatToggleRow
                key={c.chatId}
                chat={c}
                last={i === filteredChats.length - 1}
                onToggle={handleToggle}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );

  return createPortal(sheet, mountTarget);
}

interface ChatToggleRowProps {
  chat: ChatRow;
  last: boolean;
  onToggle: (chatId: number, nextValue: boolean) => void;
}

function ChatToggleRow({ chat, last, onToggle }: ChatToggleRowProps) {
  const displayName = chat.chatName.trim() || 'Без названия';
  return (
    <div style={{
      display: 'flex', alignItems: 'center', gap: 12,
      padding: '12px 14px',
      borderBottom: last ? 'none' : `0.5px solid ${T.hairline}`,
    }}>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{
          fontSize: 14.5, color: T.ink, fontWeight: 500, lineHeight: 1.25,
          overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
        }}>{displayName}</div>
      </div>
      <SwitchToggle
        value={chat.isActive}
        disabled={chat.toggling}
        onChange={(next) => onToggle(chat.chatId, next)}
        ariaLabel={`Активировать чат ${displayName}`}
      />
    </div>
  );
}

interface SwitchToggleProps {
  value: boolean;
  disabled?: boolean;
  onChange: (v: boolean) => void;
  ariaLabel?: string;
}

function SwitchToggle({ value, disabled, onChange, ariaLabel }: SwitchToggleProps) {
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

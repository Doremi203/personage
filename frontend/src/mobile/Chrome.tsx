import type { CSSProperties, ReactNode, Ref } from 'react';
import {
  Bell,
  Calendar,
  CheckSquare,
  Search as SearchIcon,
  Settings as SettingsIcon,
  type LucideIcon,
} from 'lucide-react';
import { SANS, SERIF, T } from './tokens';

// ─── Large title HIG header ────────────────────────────────────
interface LargeHeaderProps {
  title: ReactNode;
  subtitle?: ReactNode;
  leading?: ReactNode;
  trailing?: ReactNode;
  compact?: boolean;
}

export function LargeHeader({
  title,
  subtitle,
  leading,
  trailing,
  compact = false,
}: LargeHeaderProps) {
  return (
    <div style={{
      padding: compact ? '6px 16px 8px' : '8px 20px 10px',
      background: T.bg,
    }}>
      {(leading || trailing) && (
        <div style={{
          display: 'flex', alignItems: 'center', justifyContent: 'space-between',
          minHeight: 28, marginBottom: compact ? 4 : 6,
        }}>
          <div>{leading}</div>
          <div style={{ display: 'flex', gap: 6 }}>{trailing}</div>
        </div>
      )}
      <div style={{
        fontFamily: SERIF,
        fontSize: compact ? 30 : 34,
        lineHeight: 1.05,
        color: T.ink,
        letterSpacing: -0.5,
      }}>
        {title}
      </div>
      {subtitle && (
        <div style={{
          fontSize: 13.5, color: T.ink3, marginTop: 4, lineHeight: 1.4,
        }}>{subtitle}</div>
      )}
    </div>
  );
}

// ─── Round icon button ────────────────────────────────────────
interface IconButtonProps {
  icon: LucideIcon;
  onClick?: () => void;
  badge?: number;
  ariaLabel?: string;
}

export function IconButton({ icon: Icon, onClick, badge, ariaLabel }: IconButtonProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={ariaLabel}
      style={{
        width: 36, height: 36, borderRadius: '50%',
        background: T.subtle, border: 'none',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        color: T.ink, cursor: 'pointer', position: 'relative',
        padding: 0,
      }}
    >
      <Icon size={18} strokeWidth={1.7} />
      {badge !== undefined && badge > 0 && (
        <span style={{
          position: 'absolute', top: -2, right: -2,
          minWidth: 16, height: 16, borderRadius: 999,
          background: T.danger, color: '#fff',
          fontSize: 10, fontWeight: 600,
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          padding: '0 4px', border: `2px solid ${T.bg}`,
        }}>{badge}</span>
      )}
    </button>
  );
}

// ─── iOS-style search bar ──────────────────────────────────────
interface SearchBarProps {
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
  inputRef?: Ref<HTMLInputElement>;
}

export function SearchBar({ value, onChange, placeholder = 'Поиск', inputRef }: SearchBarProps) {
  return (
    <div style={{ padding: '6px 16px 10px' }}>
      <div style={{
        display: 'flex', alignItems: 'center', gap: 8,
        background: T.subtle,
        borderRadius: 10, padding: '8px 12px',
      }}>
        <SearchIcon size={15} strokeWidth={1.8} style={{ color: T.ink3 }} />
        <input
          ref={inputRef}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={placeholder}
          style={{
            flex: 1, background: 'transparent', border: 'none', outline: 'none',
            fontFamily: SANS, fontSize: 16, color: T.ink, minWidth: 0,
          }}
        />
      </div>
    </div>
  );
}

// ─── Pill-shaped action chip (icon + label) ───────────────────
interface SearchChipProps {
  label?: string;
  onClick?: () => void;
}

export function SearchChip({ label = 'Поиск', onClick }: SearchChipProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      style={{
        display: 'flex', alignItems: 'center', gap: 6,
        background: T.subtle, border: `0.5px solid ${T.hairline}`,
        borderRadius: 999, padding: '6px 12px 6px 10px',
        color: T.ink, cursor: 'pointer',
        fontFamily: SANS, fontSize: 13, fontWeight: 500,
      }}
    >
      <SearchIcon size={15} strokeWidth={2} />
      {label}
    </button>
  );
}

// ─── Segmented control (HIG) ───────────────────────────────────
export interface SegmentedItem<T extends string> {
  id: T;
  label: string;
  count?: number;
  icon?: LucideIcon;
}

interface SegmentedProps<T extends string> {
  items: SegmentedItem<T>[];
  value: T;
  onChange: (v: T) => void;
}

export function Segmented<T extends string>({ items, value, onChange }: SegmentedProps<T>) {
  return (
    <div style={{
      display: 'flex',
      background: T.subtle,
      borderRadius: 9,
      padding: 2,
      margin: '0 16px 12px',
    }}>
      {items.map((it) => {
        const active = it.id === value;
        const Icon = it.icon;
        return (
          <button
            type="button"
            key={it.id}
            aria-pressed={active}
            onClick={() => onChange(it.id)}
            style={{
              flex: 1, padding: '7px 10px', borderRadius: 7,
              background: active ? T.surface : 'transparent',
              border: 'none', cursor: 'pointer',
              fontFamily: SANS, fontSize: 13, fontWeight: active ? 600 : 500,
              color: active ? T.ink : T.ink2,
              boxShadow: active ? '0 1px 2px rgba(0,0,0,0.06), 0 0 0 0.5px rgba(0,0,0,0.04)' : 'none',
              transition: 'background .15s, color .15s',
              display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 5,
              minWidth: 0,
            }}
          >
            {Icon && <Icon size={13} strokeWidth={2} />}
            <span style={{ overflow: 'hidden', textOverflow: 'ellipsis' }}>{it.label}</span>
            {it.count !== undefined && (
              <span style={{
                fontSize: 11, color: active ? T.ink3 : T.ink4, fontWeight: 500,
                fontVariantNumeric: 'tabular-nums',
              }}>{it.count}</span>
            )}
          </button>
        );
      })}
    </div>
  );
}

// ─── Category chip row ────────────────────────────────────────
export interface CategoryChipItem<T extends string> {
  id: T;
  label: string;
  icon?: LucideIcon;
}

interface CategoryChipsProps<T extends string> {
  items: CategoryChipItem<T>[];
  value: T;
  onChange: (v: T) => void;
}

export function CategoryChips<T extends string>({ items, value, onChange }: CategoryChipsProps<T>) {
  return (
    <div style={{
      display: 'flex',
      gap: 8,
      padding: '0 16px 12px',
      overflowX: 'auto',
      scrollbarWidth: 'none',
    }}>
      {items.map((it) => {
        const active = it.id === value;
        const Icon = it.icon;
        return (
          <button
            type="button"
            key={it.id}
            onClick={() => onChange(it.id)}
            style={{
              display: 'inline-flex', alignItems: 'center', gap: 6,
              flexShrink: 0,
              padding: '7px 14px',
              borderRadius: 999,
              cursor: 'pointer',
              fontFamily: SANS, fontSize: 13, fontWeight: active ? 600 : 500,
              background: active ? T.ink : T.surface,
              color: active ? T.surface : T.ink,
              border: active ? 'none' : `0.5px solid ${T.hairline}`,
              transition: 'background .15s, color .15s',
              letterSpacing: -0.05,
            }}
          >
            {Icon && <Icon size={14} strokeWidth={1.8} />}
            <span>{it.label}</span>
          </button>
        );
      })}
    </div>
  );
}

// ─── Bottom tab bar ────────────────────────────────────────────
export type Tab = 'tasks' | 'schedule' | 'notifications' | 'settings';

interface TabBarProps {
  value: Tab;
  onChange: (t: Tab) => void;
  badges?: Partial<Record<Tab, number>>;
}

const TAB_ITEMS: { id: Tab; label: string; icon: LucideIcon }[] = [
  { id: 'tasks',         label: 'Задачи',      icon: CheckSquare },
  { id: 'schedule',      label: 'Расписание',  icon: Calendar },
  { id: 'notifications', label: 'Уведомления', icon: Bell },
  { id: 'settings',      label: 'Настройки',   icon: SettingsIcon },
];

export function TabBar({ value, onChange, badges = {} }: TabBarProps) {
  return (
    <div style={{
      flexShrink: 0,
      borderTop: `0.5px solid ${T.hairline}`,
      background: 'rgba(255,255,255,0.92)',
      backdropFilter: 'saturate(180%) blur(20px)',
      WebkitBackdropFilter: 'saturate(180%) blur(20px)',
      paddingBottom: 22,
      paddingTop: 6,
      display: 'flex',
    }}>
      {TAB_ITEMS.map((it) => {
        const active = it.id === value;
        const Icon = it.icon;
        const badge = badges[it.id];
        return (
          <button
            type="button"
            key={it.id}
            onClick={() => onChange(it.id)}
            style={{
              flex: 1, background: 'transparent', border: 'none', cursor: 'pointer',
              display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 3,
              padding: '4px 0',
              color: active ? T.amberDp : T.ink3,
              fontFamily: SANS,
            }}
          >
            <div style={{ position: 'relative' }}>
              <Icon size={24} strokeWidth={active ? 2 : 1.7} />
              {badge !== undefined && badge > 0 && (
                <span style={{
                  position: 'absolute', top: -3, right: -7,
                  minWidth: 16, height: 16, borderRadius: 999,
                  background: T.danger, color: '#fff',
                  fontSize: 10, fontWeight: 600,
                  display: 'flex', alignItems: 'center', justifyContent: 'center',
                  padding: '0 4px',
                }}>{badge}</span>
              )}
            </div>
            <span style={{
              fontSize: 10, fontWeight: active ? 600 : 500,
              letterSpacing: -0.1,
            }}>{it.label}</span>
          </button>
        );
      })}
    </div>
  );
}

// ─── Mobile screen shell ──────────────────────────────────────
interface ShellProps {
  tab: Tab;
  onTabChange: (t: Tab) => void;
  badges?: Partial<Record<Tab, number>>;
  children: ReactNode;
  scrollPadBottom?: number;
}

export function Shell({ tab, onTabChange, badges, children, scrollPadBottom = 12 }: ShellProps) {
  return (
    <div style={{
      width: '100%', height: '100%',
      display: 'flex', flexDirection: 'column',
      background: T.bg, color: T.ink, fontFamily: SANS,
      position: 'relative',
    }}>
      <div style={{
        flex: 1, minHeight: 0, overflow: 'auto',
        paddingBottom: scrollPadBottom,
      }}>
        {children}
      </div>
      <TabBar value={tab} onChange={onTabChange} badges={badges} />
    </div>
  );
}

// ─── Brand mark (app icon) ─────────────────────────────────────
interface BrandMarkProps {
  size?: number;
}

export function BrandMark({ size = 56 }: BrandMarkProps) {
  return (
    <div style={{
      width: size, height: size, borderRadius: size * 0.235,
      overflow: 'hidden', flexShrink: 0,
      boxShadow: '0 1px 0 rgba(255,255,255,0.4) inset, 0 6px 16px -8px rgba(0,0,0,0.25)',
    }}>
      <img src="/app-icon.png" alt="" style={{ width: '100%', height: '100%', display: 'block' }} />
    </div>
  );
}

// ─── Status pill ───────────────────────────────────────────────
interface PillProps {
  fill: string;
  ink: string;
  dot?: string;
  children: ReactNode;
  size?: 'sm' | 'md';
}

export function Pill({ fill, ink, dot, children, size = 'sm' }: PillProps) {
  const padding: CSSProperties['padding'] = size === 'sm' ? '3px 8px' : '4px 10px';
  const fontSize = size === 'sm' ? 11 : 12;
  return (
    <span style={{
      display: 'inline-flex', alignItems: 'center', gap: 5,
      padding, borderRadius: 999,
      background: fill, color: ink,
      fontSize, fontWeight: 500,
      letterSpacing: -0.05,
    }}>
      {dot && <span style={{ width: 5, height: 5, borderRadius: '50%', background: dot }} />}
      {children}
    </span>
  );
}

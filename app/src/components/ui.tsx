import type { CSSProperties, ReactNode } from 'react';

import { useTheme } from '@/theme';

export function Button({
  title,
  onPress,
  disabled,
  loading,
  variant = 'primary',
  style,
}: {
  title: string;
  onPress: () => void;
  disabled?: boolean;
  loading?: boolean;
  variant?: 'primary' | 'ghost';
  style?: CSSProperties;
}) {
  const t = useTheme();
  const primary = variant === 'primary';
  const isDisabled = !!disabled || !!loading;
  return (
    <button
      type="button"
      aria-label={title}
      aria-disabled={isDisabled}
      onClick={onPress}
      disabled={isDisabled}
      style={{
        ...buttonStyle,
        backgroundColor: primary ? t.accent : 'transparent',
        borderColor: primary ? t.accent : t.border,
        opacity: isDisabled ? 0.5 : 1,
        cursor: isDisabled ? 'default' : 'pointer',
        ...style,
      }}>
      {loading ? (
        <span className="spinner" style={{ color: primary ? t.accentText : t.text }} />
      ) : (
        <span style={{ color: primary ? t.accentText : t.text, fontWeight: 600 }}>{title}</span>
      )}
    </button>
  );
}

export function Chip({
  label,
  selected,
  onPress,
  accessibilityLabel,
}: {
  label: string;
  selected?: boolean;
  onPress?: () => void;
  accessibilityLabel?: string;
}) {
  const t = useTheme();
  return (
    <button
      type="button"
      aria-label={accessibilityLabel ?? label}
      aria-pressed={!!selected}
      onClick={onPress}
      style={{
        ...chipStyle,
        borderColor: selected ? t.accent : t.border,
        backgroundColor: selected ? t.accent : t.surfaceAlt,
        cursor: onPress ? 'pointer' : 'default',
      }}>
      <span
        style={{
          color: selected ? t.accentText : t.text,
          fontSize: 13,
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          whiteSpace: 'nowrap',
        }}>
        {label}
      </span>
    </button>
  );
}

export function Card({ children, style }: { children: ReactNode; style?: CSSProperties }) {
  const t = useTheme();
  return <div style={{ ...cardStyle, backgroundColor: t.surface, borderColor: t.border, ...style }}>{children}</div>;
}

export function ErrorText({ children }: { children: ReactNode }) {
  const t = useTheme();
  return (
    <span role="alert" style={{ color: t.danger }}>
      {children}
    </span>
  );
}

const buttonStyle: CSSProperties = {
  padding: '11px 16px',
  borderRadius: 8,
  borderWidth: 1,
  borderStyle: 'solid',
  display: 'inline-flex',
  alignItems: 'center',
  justifyContent: 'center',
  font: 'inherit',
};

const chipStyle: CSSProperties = {
  padding: '5px 10px',
  borderRadius: 14,
  borderWidth: 1,
  borderStyle: 'solid',
  marginRight: 6,
  maxWidth: 260,
  display: 'inline-flex',
  alignItems: 'center',
  font: 'inherit',
};

const cardStyle: CSSProperties = {
  borderRadius: 10,
  borderWidth: 1,
  borderStyle: 'solid',
  padding: 12,
};

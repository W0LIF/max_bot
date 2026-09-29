import type { ButtonHTMLAttributes, InputHTMLAttributes, ReactNode, TextareaHTMLAttributes } from 'react'
import { Icon } from './Icon'
import type { IconName } from './Icon'

/* ——— Логотип «Телескоп» ——— */
export function Logo({ size = 40 }: { size?: number }) {
  return (
    <span className="logo-badge" style={{ width: size, height: size }}>
      <Icon name="telescope" size={size * 0.58} />
    </span>
  )
}

/* ——— Кнопка ——— */
type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: 'primary' | 'dark' | 'ghost'
  size?: 'md' | 'sm'
  icon?: IconName
}

export function Button({ variant = 'primary', size = 'md', icon, children, className = '', ...rest }: ButtonProps) {
  return (
    <button
      type="button"
      className={`btn btn--${variant}${size === 'sm' ? ' btn--sm' : ''} ${className}`}
      {...rest}
    >
      {icon && <Icon name={icon} size={18} />}
      {children}
    </button>
  )
}

export function IconButton({ icon, label, size = 20 }: { icon: IconName; label: string; size?: number }) {
  return (
    <button type="button" className="icon-button" aria-label={label}>
      <Icon name={icon} size={size} />
    </button>
  )
}

/* ——— Карточка ——— */
export function Card({
  children,
  tone = 'paper',
  className = '',
}: {
  children: ReactNode
  tone?: 'paper' | 'flat' | 'lavender' | 'cream' | 'pink' | 'screen'
  className?: string
}) {
  const toneClass = tone === 'paper' ? '' : `card--${tone}`
  return <div className={`card ${toneClass} ${className}`}>{children}</div>
}

/* ——— Поля ввода ——— */
export function Field({ label, children }: { label?: string; children: ReactNode }) {
  return (
    <label className="field">
      {label && <span className="field__label">{label}</span>}
      {children}
    </label>
  )
}

export function Input({ icon, ...rest }: InputHTMLAttributes<HTMLInputElement> & { icon?: IconName }) {
  if (!icon) return <input className="input" {...rest} />
  return (
    <span className="input-wrap">
      <input className="input" {...rest} />
      <span className="input-wrap__icon">
        <Icon name={icon} size={19} />
      </span>
    </span>
  )
}

export function TextArea(props: TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return <textarea className="textarea" {...props} />
}

/* ——— Чипы ——— */
export function Chip({
  children,
  active,
  tone,
  onClick,
}: {
  children: ReactNode
  active?: boolean
  tone?: 'urgent' | 'important' | 'later' | 'soft'
  onClick?: () => void
}) {
  const cls = ['chip', tone ? `chip--${tone}` : '', active ? 'chip--active' : ''].join(' ')
  return (
    <button type="button" className={cls} onClick={onClick}>
      {children}
    </button>
  )
}

/* ——— Сегментированный переключатель ——— */
export function Segmented<T extends string>({
  options,
  value,
  onChange,
}: {
  options: { value: T; label: string }[]
  value: T
  onChange: (next: T) => void
}) {
  return (
    <div className="segmented">
      {options.map((option) => (
        <button
          key={option.value}
          type="button"
          className={`segmented__btn${option.value === value ? ' segmented__btn--active' : ''}`}
          onClick={() => onChange(option.value)}
        >
          {option.label}
        </button>
      ))}
    </div>
  )
}

/* ——— Переключатель ——— */
export function Switch({ checked, onChange }: { checked: boolean; onChange?: (next: boolean) => void }) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      className={`switch${checked ? ' switch--on' : ''}`}
      onClick={() => onChange?.(!checked)}
    >
      <span className="switch__thumb" />
    </button>
  )
}

/* ——— Чекбокс ——— */
export function Checkbox({
  checked,
  onChange,
  label,
  round,
}: {
  checked: boolean
  onChange?: () => void
  label?: string
  round?: boolean
}) {
  return (
    <button type="button" className="check" onClick={onChange}>
      <span className={`check__box${round ? ' check__box--round' : ''}${checked ? ' check__box--on' : ''}`}>
        {checked && <Icon name="check" size={14} strokeWidth={2.4} />}
      </span>
      {label && <span>{label}</span>}
    </button>
  )
}

/* ——— Индикаторы ——— */
export function StatusDot({ color }: { color: string }) {
  return <span className="status-dot" style={{ background: color }} />
}

export function Meter({ value, max, warn }: { value: number; max: number; warn?: boolean }) {
  return (
    <span className="meter">
      {Array.from({ length: max }, (_, i) => (
        <span
          key={i}
          className={`meter__cell${i < value ? (warn ? ' meter__cell--warn' : ' meter__cell--on') : ''}`}
        />
      ))}
    </span>
  )
}

export function ProgressRing({ value, size = 42 }: { value: number; size?: number }) {
  const r = size / 2 - 4
  const c = 2 * Math.PI * r
  return (
    <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`} className="balance__ring">
      <circle cx={size / 2} cy={size / 2} r={r} stroke="rgba(11,25,86,.14)" strokeWidth="5" fill="none" />
      <circle
        cx={size / 2}
        cy={size / 2}
        r={r}
        stroke="var(--navy)"
        strokeWidth="5"
        strokeLinecap="round"
        fill="none"
        strokeDasharray={`${(c * value) / 100} ${c}`}
        transform={`rotate(-90 ${size / 2} ${size / 2})`}
      />
      <text
        x="50%"
        y="53%"
        dominantBaseline="middle"
        textAnchor="middle"
        fontSize={size * 0.26}
        fontWeight={800}
        fill="var(--navy)"
      >
        {value}
      </text>
    </svg>
  )
}

/* ——— Рожица настроения ——— */
export type MoodValue = 'good' | 'ok' | 'bad'

const MOUTHS: Record<MoodValue, string> = {
  good: 'M8.4 13.8c1 1.4 2.2 2.1 3.6 2.1s2.6-.7 3.6-2.1',
  ok: 'M8.6 15.4h6.8',
  bad: 'M8.4 15.9c1-1.4 2.2-2.1 3.6-2.1s2.6.7 3.6 2.1',
}

export function MoodFace({ mood, size = 28 }: { mood: MoodValue; size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.6} strokeLinecap="round">
      <circle cx="12" cy="12" r="9" />
      <path d="M9 10h.01M15 10h.01" strokeWidth={2.2} />
      <path d={MOUTHS[mood]} />
    </svg>
  )
}

/* ——— Аватар ——— */
export function Avatar({ name, tone = 'blue', size = 40 }: { name: string; tone?: 'blue' | 'lavender' | 'cream' | 'pink' | 'navy'; size?: number }) {
  const initials = name
    .split(' ')
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase() ?? '')
    .join('')
  const cls = tone === 'blue' ? '' : ` avatar--${tone}`
  return (
    <span className={`avatar${cls}`} style={{ width: size, height: size, fontSize: size * 0.35 }}>
      {initials}
    </span>
  )
}

/* ——— Строка списка ——— */
export function Row({
  icon,
  iconTone,
  title,
  subtitle,
  right,
  onClick,
}: {
  icon?: IconName
  iconTone?: 'blue' | 'lavender' | 'cream'
  title: string
  subtitle?: string
  right?: ReactNode
  onClick?: () => void
}) {
  const content = (
    <>
      {icon && (
        <span className={`row__icon${iconTone ? ` row__icon--${iconTone}` : ''}`}>
          <Icon name={icon} size={18} />
        </span>
      )}
      <span className="stack grow">
        <span className="h3">{title}</span>
        {subtitle && <span className="caption">{subtitle}</span>}
      </span>
      {right ?? <Icon name="forward" size={18} className="row__chevron" />}
    </>
  )

  return onClick ? (
    <button type="button" className="row">
      {content}
    </button>
  ) : (
    <div className="row">{content}</div>
  )
}

/* ——— Заголовок секции ——— */
export function SectionTitle({ children, right }: { children: ReactNode; right?: ReactNode }) {
  return (
    <div className="section-title">
      <h2 className="h3">{children}</h2>
      {right}
    </div>
  )
}

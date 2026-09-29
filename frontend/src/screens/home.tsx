import { useState } from 'react'
import type { ReactNode } from 'react'
import {
  Avatar,
  Button,
  Card,
  Chip,
  Field,
  Input,
  Meter,
  MoodFace,
  ProgressRing,
  SectionTitle,
  Switch,
  TextArea,
} from '../ui/kit'
import type { MoodValue } from '../ui/kit'
import { Icon } from '../ui/Icon'
import { useNav } from '../nav'
import { useStore } from '../store'
import { MOOD_META, OVERDUE_COLOR, STATUS_META } from '../data'
import { isOverdue, type Task } from '../types'

/** Карточка задачи: главная, список задач, групповые задания. */
export function TaskCard({
  task,
  onToggle,
  right,
}: {
  task: Task
  onToggle?: () => void
  right?: ReactNode
}) {
  const overdue = isOverdue(task)
  const status = STATUS_META[task.status]
  const color = overdue ? OVERDUE_COLOR : status.color
  const cls = [
    'task-card',
    task.status === 'done' ? 'task-card--done' : '',
    overdue ? 'task-card--overdue' : '',
  ].join(' ')

  return (
    <div className={cls}>
      <span className="task-card__flag" style={{ background: color }} />
      <button type="button" className="check" onClick={onToggle} aria-label="Отметить выполненной">
        <span
          className={`check__box check__box--round${task.status === 'done' ? ' check__box--on' : ''}`}
          style={task.status === 'done' ? { background: color, borderColor: color } : undefined}
        >
          {task.status === 'done' && <Icon name="check" size={14} strokeWidth={2.6} />}
        </span>
      </button>
      <span className="stack grow">
        <span className="task-card__title">{task.title}</span>
        <span className="task-card__meta">
          {task.subject} · {task.deadline}
        </span>
      </span>
      {right ?? <span className="status-dot" style={{ background: color }} />}
    </div>
  )
}

/* 4-5. Главная: «Твой баланс» и задачи на сегодня */
export function HomeScreen() {
  const { push } = useNav()
  const { user, todayTasks, load, progress, mood, toggleTask } = useStore()
  const overloaded = load >= 3
  const date = new Date().toLocaleDateString('ru-RU', { weekday: 'long', day: 'numeric', month: 'long' })

  return (
    <div className="screen screen--padded">
      <div className="screen__hero">
        <h1 className="h1">Привет, {user.name}!</h1>
        <span className="caption">{date}</span>
      </div>

      <Card tone={overloaded ? 'pink' : 'lavender'} className="balance">
        <div className="row-between">
          <h2 className="h3">Твой баланс</h2>
          <ProgressRing value={progress} />
        </div>
        <div className="balance__row">
          <span className="balance__icon">
            <Icon name="flag" size={15} />
          </span>
          Нагрузка {load}/3
          <Meter value={load} max={3} warn={overloaded} />
        </div>
        <div className="balance__row">
          <span className="balance__icon">
            <Icon name="mood" size={15} />
          </span>
          Настроение: {MOOD_META[mood].label}
        </div>
        <div className="balance__row">
          <span className="balance__icon">
            <Icon name="target" size={15} />
          </span>
          Прогресс {progress}%
        </div>
        {overloaded && (
          <p className="caption">
            Нагрузка выше нормы — разгрузи день или поставь одну задачу на паузу.
          </p>
        )}
      </Card>

      <SectionTitle
        right={
          <button type="button" className="link-button" onClick={() => push('tasks')}>
            все задачи
          </button>
        }
      >
        Список задач
      </SectionTitle>

      <div className="stack" style={{ gap: 10 }}>
        {todayTasks.map((task) => (
          <TaskCard key={task.id} task={task} onToggle={() => toggleTask(task.id)} />
        ))}
        {todayTasks.length === 0 && (
          <Card tone="cream">
            <p className="body">Всё сделано — можно выдохнуть.</p>
          </Card>
        )}
      </div>

      <button type="button" className="fab" onClick={() => push('add-task')} aria-label="Добавить задачу">
        <Icon name="plus" size={24} strokeWidth={2.2} />
      </button>
    </div>
  )
}

/* 9. Настроение: статистика недели */
export function MoodScreen() {
  const { push } = useNav()
  const { state, mood } = useStore()
  const entries = state.moods.slice(-7)
  const total = entries.length || 1
  const counts = { good: 0, ok: 0, bad: 0 } as Record<MoodValue, number>
  entries.forEach((entry) => {
    counts[entry.mood] += 1
  })
  const max = Math.max(...entries.map((e) => e.value), 20)
  const x = (i: number) => 8 + (i * 84) / Math.max(entries.length - 1, 1)
  const y = (value: number) => 46 - (value / max) * 36

  return (
    <div className="screen screen--padded">
      <div className="screen__hero">
        <h1 className="h1">Общее настроение</h1>
        <span className="caption">Статистика за неделю</span>
      </div>

      <Card tone="cream">
        <div className="mood-hero">
          <span className="mood-face">
            <MoodFace mood={mood} size={30} />
          </span>
          <span className="stack">
            <span className="h2">{MOOD_META[mood].label}</span>
            <span className="caption">сегодня</span>
          </span>
        </div>
      </Card>

      <Card>
        <div className="chart">
          <svg className="chart__svg" viewBox="0 0 100 52" preserveAspectRatio="none" height="130">
            {[10, 24, 38].map((line) => (
              <line key={line} x1="4" y1={line} x2="96" y2={line} stroke="rgba(11,25,86,.08)" strokeWidth="0.6" />
            ))}
            <polyline
              points={entries.map((entry, i) => `${x(i)},${y(entry.value)}`).join(' ')}
              fill="none"
              stroke="var(--blue)"
              strokeWidth="1.4"
              strokeLinecap="round"
            />
            {entries.map((entry, i) => (
              <circle
                key={entry.id}
                cx={x(i)}
                cy={y(entry.value)}
                r="2.4"
                fill={MOOD_META[entry.mood].color}
                stroke="#fff"
                strokeWidth="1"
              />
            ))}
          </svg>
          <div className="chart__axis">
            {entries.map((entry) => (
              <span key={entry.id}>{entry.day}</span>
            ))}
          </div>
        </div>
      </Card>

      <Card>
        <div className="legend">
          {(['good', 'ok', 'bad'] as MoodValue[]).map((key) => (
            <div className="legend__item" key={key}>
              <span className="legend__dot" style={{ background: MOOD_META[key].color }} />
              {MOOD_META[key].label}
              <span className="legend__value">{Math.round((counts[key] / total) * 100)}%</span>
            </div>
          ))}
        </div>
      </Card>

      <Button onClick={() => push('mood-check')}>Сменить настроение</Button>
    </div>
  )
}

/* 10. Отметка состояния */
export function MoodCheckScreen() {
  const { back } = useNav()
  const { saveMood } = useStore()
  const [value, setValue] = useState<MoodValue>('good')
  const [note, setNote] = useState('')

  return (
    <div className="screen">
      <div className="screen__hero">
        <h1 className="h1">Как ты себя чувствуешь?</h1>
        <span className="caption">Выбери вариант или опиши своё состояние в нескольких словах</span>
      </div>

      <div className="stack" style={{ gap: 12 }}>
        {(['good', 'ok', 'bad'] as MoodValue[]).map((option) => (
          <button
            key={option}
            type="button"
            className={`mood-option${value === option ? ' mood-option--active' : ''}`}
            onClick={() => setValue(option)}
          >
            <MoodFace mood={option} size={30} />
            <span className="h3">{MOOD_META[option].label}</span>
          </button>
        ))}
      </div>

      <Field label="Хочу записать, что именно чувствую">
        <TextArea
          value={note}
          onChange={(event) => setNote(event.target.value)}
          placeholder="Например: устал после пар"
        />
      </Field>

      <div className="screen__spacer" />

      <Button
        onClick={() => {
          saveMood(value, note)
          back()
        }}
      >
        Сохранить
      </Button>
    </div>
  )
}

/* 12. Оставить заметку */
export function NoteScreen() {
  const { back } = useNav()
  const { addNote } = useStore()
  const [text, setText] = useState('')
  const [allDay, setAllDay] = useState(true)
  const [start, setStart] = useState('')
  const [end, setEnd] = useState('')

  return (
    <div className="screen">
      <div className="screen__hero">
        <h1 className="h1">Заметка</h1>
        <span className="caption">Короткое напоминание, которое будет под рукой</span>
      </div>

      <TextArea
        value={text}
        onChange={(event) => setText(event.target.value)}
        placeholder="Введите…"
      />

      <Card>
        <div className="row-between">
          <span className="stack">
            <span className="h3">Весь день</span>
            <span className="caption">Заметка висит на весь день</span>
          </span>
          <Switch checked={allDay} onChange={setAllDay} />
        </div>
      </Card>

      <div className="note-days">
        <Field label="Начало">
          <Input
            value={start}
            onChange={(event) => setStart(event.target.value)}
            placeholder="Введите…"
            disabled={allDay}
          />
        </Field>
        <Field label="Конец">
          <Input
            value={end}
            onChange={(event) => setEnd(event.target.value)}
            placeholder="Введите…"
            disabled={allDay}
          />
        </Field>
      </div>

      <div className="chip-row">
        {['Идея', 'Важно', 'Позже'].map((tag) => (
          <Chip key={tag} tone="important">
            {tag}
          </Chip>
        ))}
      </div>

      <div className="screen__spacer" />

      <Button
        disabled={!text.trim()}
        onClick={() => {
          addNote({ text: text.trim(), allDay, start, end })
          back()
        }}
      >
        Отправить
      </Button>
    </div>
  )
}

/** Аватар + имя для шапки меню. */
export function ProfileLine() {
  const { user } = useStore()
  return (
    <div className="profile">
      <Avatar name={user.name} size={44} />
      <span className="stack">
        <span className="h2">{user.name}</span>
        <span className="caption">
          {user.course} · {user.group}
        </span>
      </span>
    </div>
  )
}

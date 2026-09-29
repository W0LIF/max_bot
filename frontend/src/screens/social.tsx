import { useState } from 'react'
import { Avatar, Button, Card, Chip, Input, SectionTitle } from '../ui/kit'
import { Icon } from '../ui/Icon'
import { useNav } from '../nav'
import { useStore } from '../store'
import { ACHIEVEMENTS, STATUS_META } from '../data'
import { parseDeadline } from '../types'
import { TaskCard } from './home'

/* 13. Совместный режим: выбор однокурсников */
export function GroupScreen() {
  const { push } = useNav()
  const { state, toggleMember } = useStore()
  const [query, setQuery] = useState('')

  const members = state.groupMembers.filter((m) => m.name.toLowerCase().includes(query.toLowerCase()))

  return (
    <div className="screen">
      <div className="screen__hero">
        <h1 className="h1">Общие задачи</h1>
        <span className="caption">Участники вашей учебной группы</span>
      </div>

      <span className="input-wrap input-wrap--left">
        <Input
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder={`Поиск по группе ${state.groupMembers[0]?.group ?? ''}`}
        />
        <span className="input-wrap__icon">
          <Icon name="search" size={18} />
        </span>
      </span>

      <Card>
        <div className="people">
          {members.map((member, index) => {
            const checked = state.members.includes(String(member.id))
            return (
              <button type="button" className="people__row" key={member.id} onClick={() => toggleMember(String(member.id))}>
                <Avatar
                  name={member.name}
                  tone={(['blue', 'pink', 'cream', 'lavender'] as const)[index % 4]}
                  size={38}
                />
                <span className="stack grow">
                  <span className="h3">{member.name}</span>
                  <span className="caption">Группа {member.group}</span>
                </span>
                <span className={`check__box check__box--round${checked ? ' check__box--on' : ''}`}>
                  {checked && <Icon name="check" size={14} strokeWidth={2.6} />}
                </span>
              </button>
            )
          })}
          {members.length === 0 && <p className="caption">В группе пока нет участников.</p>}
        </div>
      </Card>

      <div className="screen__spacer" />

      <Button
        disabled={state.members.length === 0}
        onClick={() => push('group-tasks')}
      >
        Общие задачи ({state.members.length})
      </Button>
    </div>
  )
}

/* 14. Групповые задания */
export function GroupTasksScreen() {
  const { push } = useNav()
  const { state } = useStore()
  const [day, setDay] = useState((new Date().getDay() + 6) % 7)

  const chosen = state.groupMembers.filter((member) => state.members.includes(String(member.id)))
  const selectedNames = new Set(chosen.map((member) => member.name))
  const visibleTasks = state.groupTasks.filter((task) => !task.owner || selectedNames.has(task.owner))
  const weekStart = new Date()
  weekStart.setDate(weekStart.getDate() - ((weekStart.getDay() + 6) % 7))
  weekStart.setHours(0, 0, 0, 0)
  const days = ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс'].map((label, index) => {
    const date = new Date(weekStart)
    date.setDate(weekStart.getDate() + index)
    const count = visibleTasks.filter((task) => {
      const deadline = parseDeadline(task.deadline)
      return deadline?.toDateString() === date.toDateString()
    }).length
    return { label, count, date }
  })
  const selectedDate = days[day]?.date
  const dayTasks = visibleTasks.filter((task) => {
    const deadline = parseDeadline(task.deadline)
    return deadline && selectedDate && deadline.toDateString() === selectedDate.toDateString()
  })
  const done = dayTasks.filter((task) => task.status === 'done')
  const inProgress = dayTasks.filter((task) => task.status !== 'done')

  return (
    <div className="screen screen--padded">
      <div className="screen__hero">
        <h1 className="h1">Группа</h1>
        <span className="caption">Общие задачи · {state.groupMembers[0]?.group ?? 'группа'}</span>
      </div>

      <div className="chip-row">
        {chosen.map((member, index) => (
          <span
            key={member.id}
            className="chip chip--soft"
            style={{ borderColor: index === chosen.length - 1 ? 'var(--navy)' : 'transparent' }}
          >
            <Avatar name={member.name} size={20} tone={(['blue', 'pink', 'cream', 'lavender'] as const)[index % 4]} />
            {member.name}
          </span>
        ))}
        {chosen.length === 0 && <Chip onClick={() => push('group')}>Выбрать участников</Chip>}
      </div>

      <div className="week-strip">
        {days.map((item, index) => (
          <button
            key={item.label}
            type="button"
            className={`week-strip__day${day === index ? ' week-strip__day--active' : ''}`}
            onClick={() => setDay(index)}
          >
            {item.label}
            <span className="week-strip__count">{item.count}</span>
          </button>
        ))}
      </div>

      <SectionTitle>Выполнено</SectionTitle>
      <div className="stack" style={{ gap: 10 }}>
        {done.map((task) => (
          <TaskCard
            key={task.id}
            task={task}
            right={
              <span className="badge">
                <Icon name="users" size={13} /> {task.owner}
              </span>
            }
          />
        ))}
      </div>

      <SectionTitle>В процессе</SectionTitle>
      <div className="stack" style={{ gap: 10 }}>
        {inProgress.map((task) => (
          <TaskCard key={task.id} task={task} />
        ))}
      </div>

      <Card tone="flat">
        <div className="row-between">
          <span className="stack">
            <span className="h3">Средний статус группы</span>
            <span className="caption">
              {done.length} из {dayTasks.length} · {STATUS_META.in_progress.label}
            </span>
          </span>
          <span className="status-dot" style={{ background: STATUS_META.done.color }} />
        </div>
      </Card>

      <button type="button" className="fab" onClick={() => push('add-task')} aria-label="Добавить задача в группу">
        <Icon name="plus" size={24} strokeWidth={2.2} />
      </button>
    </div>
  )
}

/* 15. Карта достижений */
export function MapScreen() {
  const [active, setActive] = useState<string | null>(null)

  const points = ACHIEVEMENTS
  const path = points
    .map((node, i) => `${i === 0 ? 'M' : 'L'} ${node.x} ${node.y}`)
    .join(' ')
  const opened = points.filter((node) => !node.locked).length

  return (
    <div className="screen screen--padded">
      <div className="screen__hero">
        <h1 className="h1">Карта приключений</h1>
        <span className="caption">{opened} из {points.length} станций пройдено</span>
      </div>

      <div className="map">
        <svg className="map__path" viewBox="0 0 100 100" preserveAspectRatio="none">
          <path d={path} fill="none" stroke="var(--line-strong)" strokeWidth="0.8" strokeDasharray="2.5 2.5" />
        </svg>
        {points.map((node) => (
          <button
            type="button"
            key={node.id}
            className="map__node"
            style={{ left: `${node.x}%`, top: `${node.y}%` }}
            onClick={() => setActive(active === node.id ? null : node.id)}
          >
            <span className={`map__bubble${node.locked ? ' map__bubble--locked' : ' map__bubble--done'}`}>
              <Icon name={node.locked ? 'lock' : node.icon} size={24} />
            </span>
            <span className={`map__label${node.locked ? ' map__label--locked' : ''}`}>{node.title}</span>
          </button>
        ))}
      </div>

      {active && (
        <Card tone="lavender">
          <div className="stack">
            <span className="h3">{points.find((node) => node.id === active)?.title}</span>
            <span className="caption">
              Станция открывает подсказку, как снизить нагрузку, и новый показатель баланса.
            </span>
          </div>
        </Card>
      )}

      <Card tone="cream">
        <div className="row-between">
          <span className="stack">
            <span className="h3">Ближайшая цель</span>
            <span className="caption">Закрыть 2 просроченные задачи</span>
          </span>
          <span className="status-dot" style={{ background: STATUS_META.in_progress.color }} />
        </div>
      </Card>
    </div>
  )
}

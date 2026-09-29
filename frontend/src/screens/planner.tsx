import { type ChangeEvent, useEffect, useMemo, useRef, useState } from 'react'
import { Button, Card, Chip, Field, Input, Segmented, SectionTitle, StatusDot } from '../ui/kit'
import { Icon } from '../ui/Icon'
import { useNav } from '../nav'
import { useStore } from '../store'
import { STATUS_META, SUBJECTS, PRIORITY_META } from '../data'
import { isOverdue, parseDeadline, type Priority, type TaskStatus } from '../types'
import { TaskCard } from './home'

/* 6-7. Добавить задачу вручную / по фото */
export function AddTaskScreen() {
  const { back, push } = useNav()
  const { addTask, scanDraft, setScanDraft } = useStore()
  const [mode, setMode] = useState<'manual' | 'photo'>('manual')
  const [title, setTitle] = useState(scanDraft?.title ?? '')
  const [subject, setSubject] = useState(scanDraft?.subject ?? SUBJECTS[0])
  const [deadline, setDeadline] = useState(scanDraft?.deadline ?? '')
  const [priority, setPriority] = useState<Priority>('important')

  const submit = async () => {
    if (!title.trim()) return
    const saved = await addTask({
      title: title.trim(),
      subject,
      deadline: deadline.trim(),
      priority,
    })
    if (!saved) return
    setScanDraft(null)
    back()
  }

  return (
    <div className="screen">
      <Segmented
        value={mode}
        onChange={setMode}
        options={[
          { value: 'manual', label: 'Вручную' },
          { value: 'photo', label: 'По фото' },
        ]}
      />

      {mode === 'photo' && (
        <button type="button" className="dropzone" onClick={() => push('scan')}>
          <span className="dropzone__icon">
            <Icon name="camera" size={30} />
          </span>
          <span className="stack">
            <span className="h3">Фото расписания или задания</span>
            <span className="caption">Загрузи изображение, чтобы подготовить черновик</span>
          </span>
        </button>
      )}

      <Field label="Название задачи">
        <Input
          value={title}
          onChange={(event) => setTitle(event.target.value)}
          placeholder="Введите…"
        />
      </Field>

      <Field label="Предмет">
        <Input
          value={subject}
          onChange={(event) => setSubject(event.target.value)}
          placeholder="Введите…"
          list="subjects"
        />
        <datalist id="subjects">
          {SUBJECTS.map((item) => (
            <option key={item} value={item} />
          ))}
        </datalist>
      </Field>

      <Field label="Срок">
        <Input
          value={deadline}
          onChange={(event) => setDeadline(event.target.value)}
          type="datetime-local"
          icon="calendar"
        />
      </Field>

      <Field label="Приоритет">
        <div className="chip-row">
          {(Object.keys(PRIORITY_META) as Priority[]).map((key) => (
            <Chip
              key={key}
              tone={PRIORITY_META[key].tone}
              active={priority === key}
              onClick={() => setPriority(key)}
            >
              {PRIORITY_META[key].label}
            </Chip>
          ))}
        </div>
      </Field>

      <div className="upload-row">
        <span className="upload-preview">
          <Icon name="doc" size={22} />
        </span>
        <span className="stack grow">
          <span className="h3">Файл или фото</span>
          <span className="caption">Необязательно</span>
        </span>
        <button type="button" className="icon-button" onClick={() => push('scan')} aria-label="Камера">
          <Icon name="camera" size={18} />
        </button>
      </div>

      <div className="screen__spacer" />

      <Button disabled={!title.trim() || !deadline} onClick={submit}>
        Добавить
      </Button>
    </div>
  )
}

/* 8. Распознавание */
export function ScanScreen() {
  const { push } = useNav()
  const { scanTask } = useStore()
  const fileInput = useRef<HTMLInputElement>(null)
  const [photo, setPhoto] = useState<File | null>(null)
  const [preview, setPreview] = useState<string | null>(null)
  const [scanning, setScanning] = useState(false)

  useEffect(() => () => {
    if (preview) URL.revokeObjectURL(preview)
  }, [preview])

  const choosePhoto = (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0] ?? null
    setPhoto(file)
    setPreview(file ? URL.createObjectURL(file) : null)
  }

  const scan = async () => {
    if (!photo) return
    setScanning(true)
    const ok = await scanTask(photo)
    setScanning(false)
    if (ok) push('add-task')
  }

  return (
    <div className="screen">
      <input
        ref={fileInput}
        className="visually-hidden"
        type="file"
        accept="image/*"
        capture="environment"
        onChange={choosePhoto}
      />
      <button
        type="button"
        className={`dropzone${photo ? ' dropzone--active' : ''}`}
        onClick={() => fileInput.current?.click()}
      >
        {preview ? <img className="scan-preview" src={preview} alt="Предпросмотр загруженного фото" /> : (
          <span className="dropzone__icon"><Icon name="camera" size={32} /></span>
        )}
        <span className="stack">
          <span className="h3">{photo?.name ?? 'Выберите фото задания'}</span>
          <span className="caption">Камера или изображение из галереи</span>
        </span>
      </button>

      <Card tone="screen">
        <div className="stack">
          <span className="h3">Черновик задачи</span>
          <span className="body">Фото отправляется в обработчик для подготовки формы.</span>
          <span className="caption">OCR пока не подключён. Проверьте поля перед сохранением.</span>
        </div>
      </Card>

      <div className="screen__spacer" />

      <Button disabled={!photo || scanning} icon="plus" onClick={scan}>
        {scanning ? 'Готовим черновик…' : 'Продолжить к задаче'}
      </Button>
    </div>
  )
}

/* Список всех задач с фильтром по состоянию */
export function TasksScreen() {
  const { push } = useNav()
  const { state, toggleTask } = useStore()
  const [filter, setFilter] = useState<'all' | TaskStatus | 'overdue'>('all')

  /* «Просрочено» — не статус, а вычисляемый признак по deadline. */
  const tasks =
    filter === 'all'
      ? state.tasks
      : filter === 'overdue'
        ? state.tasks.filter((t) => isOverdue(t))
        : state.tasks.filter((t) => t.status === filter)
  const filters: { value: 'all' | TaskStatus | 'overdue'; label: string }[] = [
    { value: 'all', label: 'Все' },
    { value: 'in_progress', label: 'В процессе' },
    { value: 'new', label: 'Новые' },
    { value: 'done', label: 'Готово' },
    { value: 'overdue', label: 'Просрочено' },
  ]

  return (
    <div className="screen screen--padded">
      <div className="screen__hero">
        <h1 className="h1">Задачи</h1>
        <span className="caption">Всего {state.tasks.length} · фильтруй по состоянию</span>
      </div>

      <div className="chip-row">
        {filters.map((item) => (
          <Chip key={item.value} active={filter === item.value} onClick={() => setFilter(item.value)}>
            {item.label}
          </Chip>
        ))}
      </div>

      <div className="stack" style={{ gap: 10 }}>
        {tasks.map((task) => (
          <TaskCard key={task.id} task={task} onToggle={() => toggleTask(task.id)} />
        ))}
        {tasks.length === 0 && <Card tone="cream">Нет задач в этом статусе.</Card>}
      </div>

      <button type="button" className="fab" onClick={() => push('add-task')} aria-label="Добавить задачу">
        <Icon name="plus" size={24} strokeWidth={2.2} />
      </button>
    </div>
  )
}

/* 11. Календарь */
export function CalendarScreen() {
  const { push } = useNav()
  const { state, done } = useStore()
  const [offset, setOffset] = useState(0)

  const { cells, monthTitle } = useMemo(() => {
    const now = new Date()
    const first = new Date(now.getFullYear(), now.getMonth() + offset, 1)
    const title = first.toLocaleDateString('ru-RU', { month: 'long', year: 'numeric' })
    const leading = (first.getDay() + 6) % 7
    const days = new Date(first.getFullYear(), first.getMonth() + 1, 0).getDate()
    const prevDays = new Date(first.getFullYear(), first.getMonth(), 0).getDate()
    const list: { day: number; muted: boolean; today: boolean; mark: boolean }[] = []

    for (let i = leading; i > 0; i -= 1) {
      list.push({ day: prevDays - i + 1, muted: true, today: false, mark: false })
    }
    const today = new Date()
    for (let day = 1; day <= days; day += 1) {
      list.push({
        day,
        muted: false,
        today: day === today.getDate() && offset === 0,
        mark: state.tasks.some((task) => {
          const deadline = parseDeadline(task.deadline)
          return deadline?.toDateString() === new Date(first.getFullYear(), first.getMonth(), day).toDateString()
        }),
      })
    }
    while (list.length % 7 !== 0) {
      list.push({ day: list.length - (leading + days) + 1, muted: true, today: false, mark: false })
    }
    return { cells: list, monthTitle: title }
  }, [offset, state.tasks])

  const notes = state.notes.slice(0, 3)
  const today = new Date()
  const todayTasks = state.tasks.filter((task) => parseDeadline(task.deadline)?.toDateString() === today.toDateString())
  const reminders = state.tasks.filter((task) => task.status !== 'done').slice(0, 3)

  return (
    <div className="screen screen--padded">
      <Card>
        <div className="calendar__head">
          <button type="button" className="icon-button" onClick={() => setOffset((v) => v - 1)} aria-label="Предыдущий месяц">
            <Icon name="back" size={16} />
          </button>
          <span className="h3" style={{ textTransform: 'capitalize' }}>
            {monthTitle}
          </span>
          <button type="button" className="icon-button" onClick={() => setOffset((v) => v + 1)} aria-label="Следующий месяц">
            <Icon name="forward" size={16} />
          </button>
        </div>
        <div className="calendar__grid">
          {['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс'].map((day) => (
            <span className="calendar__weekday" key={day}>
              {day}
            </span>
          ))}
          {cells.map((cell, index) => (
            <span
              key={`${cell.day}-${index}`}
              className={`calendar__day${cell.muted ? ' calendar__day--muted' : ''}${
                cell.today ? ' calendar__day--today' : ''
              }${cell.mark ? ' calendar__day--mark' : ''}`}
            >
              {cell.day}
            </span>
          ))}
        </div>
      </Card>

      <SectionTitle right={<StatusDot color="var(--st-done)" />}>Сегодня</SectionTitle>

      <Card>
        <div className="row">
          <span className="row__icon row__icon--lavender">
            <Icon name="check" size={17} />
          </span>
          <span className="stack grow">
            <span className="h3">{done} задач выполнено</span>
            <span className="caption">из {state.tasks.length} в работе</span>
          </span>
        </div>
      </Card>

      <div className="stack" style={{ gap: 10 }}>
        {todayTasks.slice(0, 2).map((task) => (
          <TaskCard key={task.id} task={task} />
        ))}
        {todayTasks.length === 0 && <p className="caption">На сегодня задач нет.</p>}
      </div>

      <SectionTitle>Напоминания</SectionTitle>
      <div className="list">
        {reminders.map((reminder) => (
          <div className="row" key={reminder.id}>
            <span className="row__icon row__icon--cream">
              <Icon name="bell" size={17} />
            </span>
            <span className="stack grow">
              <span className="h3">{reminder.title}</span>
              <span className="caption">{reminder.deadline}</span>
            </span>
            <StatusDot color={STATUS_META.new.color} />
          </div>
        ))}
        {notes.map((note) => (
          <div className="row" key={note.id}>
            <span className="row__icon">
              <Icon name="doc" size={17} />
            </span>
            <span className="stack grow">
              <span className="h3">{note.text}</span>
              <span className="caption">
                {note.allDay ? 'Весь день' : `${formatNoteTime(note.start)} — ${formatNoteTime(note.end)}`}
              </span>
            </span>
          </div>
        ))}
      </div>

      <button type="button" className="fab" onClick={() => push('note')} aria-label="Оставить заметку">
        <Icon name="plus" size={24} strokeWidth={2.2} />
      </button>
    </div>
  )
}

function formatNoteTime(value: string): string {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
}

import { useState } from 'react'
import { Avatar, Button, Card, Field, Logo, Row, SectionTitle, Switch, TextArea } from '../ui/kit'
import { Icon } from '../ui/Icon'
import { useNav } from '../nav'
import { useStore } from '../store'
import { FAQ } from '../data'
import { closeWebApp } from '../webapp'

/* 16. Меню */
export function MenuScreen() {
  const { push, openTab } = useNav()
  const { user, state } = useStore()

  return (
    <div className="screen screen--padded">
      <div className="screen__hero">
        <h1 className="h1">Меню</h1>
      </div>

      <Card>
        <div className="profile">
          <Avatar name={user.name} size={46} />
          <span className="stack grow">
            <span className="h2">{user.name}</span>
            <span className="caption">
              {[user.course, user.group].filter(Boolean).join(' · ')}
            </span>
          </span>
          <Icon name="forward" size={18} className="row__chevron" />
        </div>
      </Card>

      <div className="list">
        <Row icon="help" title="Помощь" subtitle="Частые вопросы" onClick={() => push('help')} />
        <Row icon="mail" title="Обратная связь" subtitle="Что улучшить?" onClick={() => push('feedback')} />
        <Row icon="gear" title="Настройки" subtitle="Тема, язык, уведомления" onClick={() => push('settings')} />
        <Row
          icon="users"
          iconTone="lavender"
          title="Совместный режим"
          subtitle={`${state.members.length} участников группы`}
          onClick={() => push('group')}
        />
        <Row
          icon="map"
          iconTone="cream"
          title="Карта приключений"
          subtitle="Достижения за неделю"
          onClick={() => openTab('map')}
        />
      </div>

      <Card tone="screen" className="version">
        <Logo size={40} />
        <span className="stack grow">
          <span className="h3">Телескоп</span>
          <span className="caption">Версия 1.0 · фокус на учёбу</span>
        </span>
      </Card>
    </div>
  )
}

/* 17. Помощь — FAQ-аккордеон */
export function HelpScreen() {
  const { push } = useNav()
  const [open, setOpen] = useState<string | null>(FAQ[0]?.q ?? null)

  return (
    <div className="screen">
      <div className="screen__hero">
        <h1 className="h1">Помощь</h1>
        <span className="caption">У нас есть раздел, позволяющий быстро найти ответ</span>
      </div>

      <div className="list">
        {FAQ.map((item) => {
          const expanded = open === item.q
          return (
            <div key={item.q}>
              <button type="button" className="faq__q" onClick={() => setOpen(expanded ? null : item.q)}>
                {item.q}
                <Icon name="forward" size={16} className={`faq__chevron${expanded ? ' faq__chevron--open' : ''}`} />
              </button>
              {expanded && <p className="faq__a">{item.a}</p>}
            </div>
          )
        })}
      </div>

      <Card tone="lavender">
        <div className="row-between">
          <span className="stack">
            <span className="h3">Не нашёл ответ?</span>
            <span className="caption">Сообщение сохранится для команды</span>
          </span>
          <Button size="sm" variant="dark" onClick={() => push('feedback')}>
            Написать
          </Button>
        </div>
      </Card>
    </div>
  )
}

/* 18. Обратная связь */
export function FeedbackScreen() {
  const { back } = useNav()
  const { sendFeedback } = useStore()
  const [text, setText] = useState('')
  const [sent, setSent] = useState(false)
  const [sending, setSending] = useState(false)

  return (
    <div className="screen">
      <div className="screen__hero">
        <h1 className="h1">Обратная связь ✍️</h1>
      </div>

      <span className="feedback__mail">
        <Icon name="mail" size={34} />
      </span>

      <Card tone="cream">
        <p className="body">
          У вас есть идея, предложение или вопрос? Мы всегда на связи и читаем каждое сообщение.
        </p>
      </Card>

      <Field label="Ваше сообщение">
        <TextArea
          value={text}
          onChange={(event) => setText(event.target.value)}
          placeholder="Ваше сообщение…"
        />
      </Field>

      {sent && (
        <Card tone="lavender">
          <p className="body">Спасибо! Сообщение сохранено для команды.</p>
        </Card>
      )}

      <div className="screen__spacer" />

      <Button
        disabled={!text.trim() || sent || sending}
        onClick={async () => {
          setSending(true)
          const ok = await sendFeedback(text.trim())
          setSending(false)
          if (ok) {
            setSent(true)
            setText('')
          }
        }}
      >
        {sending ? 'Отправляем…' : 'Отправить'}
      </Button>
      <button type="button" className="link-button" onClick={back}>
        Вернуться назад
      </button>
    </div>
  )
}

/* 19. Настройки */
export function SettingsScreen() {
  const { state, setSetting, reset, connection, user } = useStore()

  return (
    <div className="screen">
      <div className="screen__hero">
        <h1 className="h1">Настройки</h1>
      </div>

      <SectionTitle>Аккаунт</SectionTitle>
      <div className="list">
        <Row icon="user" title={user.name} subtitle={[user.course, user.group].filter(Boolean).join(' · ')} />
        <Row
          icon="bell"
          title="Уведомления"
          subtitle={state.settings.notifications ? 'Напоминания бота включены' : 'Напоминания бота выключены'}
          right={<Switch checked={state.settings.notifications} onChange={(v) => setSetting('notifications', v)} />}
        />
      </div>

      <SectionTitle>Приложение</SectionTitle>
      <div className="list">
        <Row
          icon={state.settings.darkTheme ? 'moon' : 'sun'}
          title="Тема"
          subtitle={state.settings.darkTheme ? 'Тёмная' : 'Светлая'}
          right={<Switch checked={state.settings.darkTheme} onChange={(v) => setSetting('darkTheme', v)} />}
        />
        <Row icon="globe" title="Язык" subtitle="Русский" />
        <Row icon="shield-check" title="Безопасность" subtitle="Данные связаны с аккаунтом MAX" />
      </div>

      {connection === 'demo' && (
        <Card tone="flat">
          <div className="row-between">
            <span className="stack">
              <span className="h3">Сбросить демо-данные</span>
              <span className="caption">Вернуть задачи и настроение как в макете</span>
            </span>
            <Button size="sm" variant="ghost" onClick={reset}>
              сброс
            </Button>
          </div>
        </Card>
      )}

      <div className="screen__spacer" />

      <button
        type="button"
        className="link-button"
        onClick={closeWebApp}
      >
        Закрыть приложение
      </button>
    </div>
  )
}

import { useState } from 'react'
import { Button, Card, Checkbox, Logo } from '../ui/kit'
import { Icon } from '../ui/Icon'
import { useNav } from '../nav'
import { useStore } from '../store'

/* 1. Политика конфиденциальности */
export function PrivacyScreen() {
  const { replace } = useNav()
  const { acceptConsent } = useStore()
  const [agreed, setAgreed] = useState(false)

  return (
    <div className="screen onboarding">
      <span className="onboarding__shield">
        <Icon name="shield" size={44} />
      </span>

      <div className="stack">
        <h1 className="h2">Политика конфиденциальности</h1>
        <div className="onboarding__legal">
          <p className="body">
            Мы заботимся о вашей приватности. «Телескоп» обрабатывает только те данные, которые
            необходимы для работы приложения: задачи, расписание, статистика и настройки.
          </p>
          <p className="caption">
            Данные хранятся на вашем устройстве и не передаются третьим лицам.
          </p>
        </div>
      </div>

      <div className="screen__spacer" />

      <div className="screen__footer">
        <Checkbox
          checked={agreed}
          round={false}
          label="Я согласен с политикой конфиденциальности"
          onChange={() => setAgreed((v) => !v)}
        />
        <Button
          disabled={!agreed}
          onClick={() => {
            acceptConsent()
            replace('auth')
          }}
        >
          Продолжить
        </Button>
      </div>
    </div>
  )
}

/* 2. Вход / регистрация */
export function AuthScreen() {
  const { replace } = useNav()
  const { authorize } = useStore()

  const login = () => {
    authorize()
    replace('auth-ok')
  }

  return (
    <div className="screen onboarding">
      <div className="onboarding__brand">
        <Logo size={48} />
        <div className="stack" style={{ alignItems: 'flex-start' }}>
          <h1 className="h2">Телескоп</h1>
          <span className="onboarding__tagline">фокус на учёбу · фокус на себе</span>
        </div>
      </div>

      <div className="onboarding__devices">
        <Icon name="phone" size={44} />
        <Icon name="globe" size={34} />
      </div>

      <Card tone="screen" className="center">
        <p className="body">
          Отмечайте задачи, следите за нагрузкой и настроением — всё в одном месте.
        </p>
      </Card>

      <div className="screen__spacer" />

      <div className="screen__footer">
        <Button onClick={login}>Войти через MAX</Button>
        <Button variant="ghost" icon="phone" onClick={login}>
          Войти по номеру телефона
        </Button>
      </div>
    </div>
  )
}

/* 3. Подтверждение / переход */
export function AuthOkScreen() {
  const { openTab } = useNav()

  return (
    <div className="screen onboarding">
      <span className="onboarding__shield onboarding__shield--ok">
        <Icon name="check" size={46} strokeWidth={2.2} />
      </span>

      <div className="stack">
        <h1 className="h2">Отлично!</h1>
        <p className="h3">Вы успешно авторизованы</p>
        <p className="body muted">
          Теперь вы можете перейти в мини-приложение «Телескоп» и работать с задачами в этом чате.
        </p>
      </div>

      <div className="screen__spacer" />

      <div className="screen__footer">
        <Button onClick={() => openTab('home')}>Открыть приложение</Button>
        <button type="button" className="link-button" onClick={() => openTab('home')}>
          Позже
        </button>
      </div>
    </div>
  )
}

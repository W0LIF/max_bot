import { useEffect, useState } from 'react'
import { Button, Panel, Typography, Container, Flex, Grid } from '@maxhub/max-ui'

interface MaxUser {
  id: number
  first_name?: string
  last_name?: string
}

function App() {
  const [user, setUser] = useState<MaxUser | null>(null)
  const [status, setStatus] = useState('')
  const [isLoading, setIsLoading] = useState(false)

  useEffect(() => {
    const webApp = (window as any).WebApp
    if (webApp?.initDataUnsafe?.user) {
      setUser(webApp.initDataUnsafe.user)
    } else {
      setUser({ id: 0, first_name: 'Разработчик (браузер)' })
    }
  }, [])

  const checkBackend = async () => {
    setIsLoading(true)
    setStatus('')
    try {
      const webApp = (window as any).WebApp
      const initData = webApp?.initData || ''

      const response = await fetch('/api/validate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ initData }),
      })
      const result = await response.json()
      setStatus(result.message)
    } catch {
      setStatus('Ошибка соединения с сервером')
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <Panel mode="secondary" style={{ minHeight: '100vh', padding: 16 }}>
      <Container>
        <Flex direction="column" gap={16}>
          <Typography.Title>StudyFlow</Typography.Title>
          <Typography.Text>
            Привет, {user?.first_name || 'гость'}!
          </Typography.Text>
          <Grid gap={12} cols={1}>
            <Button onClick={checkBackend} loading={isLoading}>
              Проверить связь с сервером
            </Button>
          </Grid>
          {status && (
            <Panel mode="primary" style={{ padding: 12 }}>
              <Typography.Text>{status}</Typography.Text>
            </Panel>
          )}
        </Flex>
      </Container>
    </Panel>
  )
}

export default App
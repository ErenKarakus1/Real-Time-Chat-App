import { useEffect, useMemo, useState, type FormEvent } from 'react'
import './App.css'
import { ApiError, login, me, register, type User } from './api'

const TOKEN_KEY = 'rtc_token'

type AuthMode = 'login' | 'register'

type AuthForm = {
  username: string
  email: string
  password: string
}

const emptyForm: AuthForm = {
  username: '',
  email: '',
  password: '',
}

function getStoredToken() {
  return localStorage.getItem(TOKEN_KEY)
}

function App() {
  const [mode, setMode] = useState<AuthMode>('login')
  const [form, setForm] = useState<AuthForm>(emptyForm)
  const [token, setToken] = useState(() => getStoredToken())
  const [user, setUser] = useState<User | null>(null)
  const [isBootstrapping, setIsBootstrapping] = useState(Boolean(token))
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')

  useEffect(() => {
    if (!token) {
      return
    }

    let isCurrent = true

    me(token)
      .then((currentUser) => {
        if (isCurrent) {
          setUser(currentUser)
        }
      })
      .catch(() => {
        if (isCurrent) {
          localStorage.removeItem(TOKEN_KEY)
          setToken(null)
          setUser(null)
        }
      })
      .finally(() => {
        if (isCurrent) {
          setIsBootstrapping(false)
        }
      })

    return () => {
      isCurrent = false
    }
  }, [token])

  const title = useMemo(() => {
    if (isBootstrapping) {
      return 'Opening your workspace'
    }

    return mode === 'login' ? 'Welcome back' : 'Create your account'
  }, [isBootstrapping, mode])

  function updateField(field: keyof AuthForm, value: string) {
    setForm((currentForm) => ({
      ...currentForm,
      [field]: value,
    }))
  }

  function switchMode(nextMode: AuthMode) {
    setMode(nextMode)
    setError('')
    setNotice('')
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError('')
    setNotice('')
    setIsSubmitting(true)

    try {
      if (mode === 'register') {
        await register({
          username: form.username.trim(),
          email: form.email.trim(),
          password: form.password,
        })
        setMode('login')
        setForm((currentForm) => ({ ...currentForm, password: '' }))
        setNotice('Account created. Sign in with your new credentials.')
        return
      }

      const result = await login({
        email: form.email.trim(),
        password: form.password,
      })
      localStorage.setItem(TOKEN_KEY, result.token)
      setToken(result.token)
      setUser(result.user)
      setForm(emptyForm)
    } catch (caughtError) {
      setError(
        caughtError instanceof ApiError
          ? caughtError.message
          : 'Something went wrong. Try again.',
      )
    } finally {
      setIsSubmitting(false)
    }
  }

  function handleLogout() {
    localStorage.removeItem(TOKEN_KEY)
    setToken(null)
    setUser(null)
    setForm(emptyForm)
    setMode('login')
  }

  if (isBootstrapping) {
    return (
      <main className="auth-page">
        <section className="auth-panel" aria-busy="true">
          <p className="eyebrow">Real Time Chat</p>
          <h1>{title}</h1>
          <div className="loading-bar" aria-hidden="true" />
        </section>
      </main>
    )
  }

  if (user) {
    return (
      <main className="app-shell">
        <aside className="sidebar" aria-label="Conversations">
          <div className="brand-row">
            <span className="brand-mark" aria-hidden="true">
              R
            </span>
            <span>Real Time Chat</span>
          </div>
          <button className="new-chat-button" type="button">
            New chat
          </button>
          <div className="empty-list">No conversations yet</div>
        </aside>

        <section className="chat-stage" aria-label="Chat">
          <header className="topbar">
            <div>
              <p className="eyebrow">Signed in</p>
              <h1>{user.username}</h1>
            </div>
            <button className="ghost-button" type="button" onClick={handleLogout}>
              Sign out
            </button>
          </header>

          <div className="empty-chat">
            <p className="eyebrow">Ready</p>
            <h2>Conversations land here next.</h2>
          </div>
        </section>
      </main>
    )
  }

  return (
    <main className="auth-page">
      <section className="auth-panel">
        <div>
          <p className="eyebrow">Real Time Chat</p>
          <h1>{title}</h1>
        </div>

        <div className="mode-switch" role="tablist" aria-label="Authentication mode">
          <button
            type="button"
            role="tab"
            aria-selected={mode === 'login'}
            onClick={() => switchMode('login')}
          >
            Login
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={mode === 'register'}
            onClick={() => switchMode('register')}
          >
            Register
          </button>
        </div>

        <form className="auth-form" onSubmit={handleSubmit}>
          {mode === 'register' && (
            <label>
              Username
              <input
                autoComplete="username"
                minLength={2}
                onChange={(event) => updateField('username', event.target.value)}
                required
                type="text"
                value={form.username}
              />
            </label>
          )}

          <label>
            Email
            <input
              autoComplete="email"
              onChange={(event) => updateField('email', event.target.value)}
              required
              type="email"
              value={form.email}
            />
          </label>

          <label>
            Password
            <input
              autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
              minLength={8}
              onChange={(event) => updateField('password', event.target.value)}
              required
              type="password"
              value={form.password}
            />
          </label>

          {error && <p className="form-message error">{error}</p>}
          {notice && <p className="form-message success">{notice}</p>}

          <button className="primary-button" disabled={isSubmitting} type="submit">
            {isSubmitting ? 'Please wait' : mode === 'login' ? 'Login' : 'Create account'}
          </button>
        </form>
      </section>
    </main>
  )
}

export default App

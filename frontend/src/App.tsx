import { useEffect, useMemo, useState, type FormEvent } from 'react'
import './App.css'
import {
  ApiError,
  createDirect,
  createRoom,
  listConversations,
  login,
  me,
  register,
  type Conversation,
  type User,
} from './api'

const TOKEN_KEY = 'rtc_token'

type AuthMode = 'login' | 'register'

type AuthForm = {
  username: string
  email: string
  password: string
}

type NewConversationForm = {
  mode: Conversation['type']
  roomName: string
  otherUserID: string
}

const emptyForm: AuthForm = {
  username: '',
  email: '',
  password: '',
}

const emptyNewConversationForm: NewConversationForm = {
  mode: 'direct',
  roomName: '',
  otherUserID: '',
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
  const [conversations, setConversations] = useState<Conversation[]>([])
  const [selectedConversationID, setSelectedConversationID] = useState('')
  const [isLoadingConversations, setIsLoadingConversations] = useState(false)
  const [conversationError, setConversationError] = useState('')
  const [newConversationForm, setNewConversationForm] = useState(
    emptyNewConversationForm,
  )
  const [isCreatingConversation, setIsCreatingConversation] = useState(false)

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

  useEffect(() => {
    if (!token || !user) {
      return
    }

    let isCurrent = true

    Promise.resolve()
      .then(() => {
        if (isCurrent) {
          setIsLoadingConversations(true)
          setConversationError('')
        }

        return listConversations(token)
      })
      .then((items) => {
        if (isCurrent) {
          setConversations(items)
          setSelectedConversationID((currentID) => currentID || items[0]?.id || '')
        }
      })
      .catch((caughtError) => {
        if (isCurrent) {
          setConversationError(
            caughtError instanceof ApiError
              ? caughtError.message
              : 'Could not load conversations.',
          )
        }
      })
      .finally(() => {
        if (isCurrent) {
          setIsLoadingConversations(false)
        }
      })

    return () => {
      isCurrent = false
    }
  }, [token, user])

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
    setConversations([])
    setSelectedConversationID('')
    setNewConversationForm(emptyNewConversationForm)
    setConversationError('')
  }

  function conversationLabel(conversation: Conversation) {
    if (conversation.name) {
      return conversation.name
    }

    return conversation.type === 'direct' ? 'Direct message' : 'Room'
  }

  function updateNewConversationForm(
    field: keyof NewConversationForm,
    value: string,
  ) {
    setNewConversationForm((currentForm) => ({
      ...currentForm,
      [field]: value,
    }))
  }

  async function handleCreateConversation(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!token) {
      return
    }

    setIsCreatingConversation(true)
    setConversationError('')

    try {
      const conversation =
        newConversationForm.mode === 'room'
          ? await createRoom(token, { name: newConversationForm.roomName.trim() })
          : await createDirect(token, {
              other_user_id: newConversationForm.otherUserID.trim(),
            })

      setConversations((currentConversations) => {
        const withoutDuplicate = currentConversations.filter(
          (item) => item.id !== conversation.id,
        )
        return [conversation, ...withoutDuplicate]
      })
      setSelectedConversationID(conversation.id)
      setNewConversationForm(emptyNewConversationForm)
    } catch (caughtError) {
      setConversationError(
        caughtError instanceof ApiError
          ? caughtError.message
          : 'Could not create conversation.',
      )
    } finally {
      setIsCreatingConversation(false)
    }
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
    const selectedConversation = conversations.find(
      (conversation) => conversation.id === selectedConversationID,
    )

    return (
      <main className="app-shell">
        <aside className="sidebar" aria-label="Conversations">
          <div className="brand-row">
            <span className="brand-mark" aria-hidden="true">
              R
            </span>
            <span>Real Time Chat</span>
          </div>
          <form className="new-chat-form" onSubmit={handleCreateConversation}>
            <div className="mode-switch compact" role="tablist" aria-label="New chat type">
              <button
                type="button"
                role="tab"
                aria-selected={newConversationForm.mode === 'direct'}
                onClick={() => updateNewConversationForm('mode', 'direct')}
              >
                DM
              </button>
              <button
                type="button"
                role="tab"
                aria-selected={newConversationForm.mode === 'room'}
                onClick={() => updateNewConversationForm('mode', 'room')}
              >
                Room
              </button>
            </div>

            {newConversationForm.mode === 'room' ? (
              <label>
                Room name
                <input
                  minLength={2}
                  onChange={(event) =>
                    updateNewConversationForm('roomName', event.target.value)
                  }
                  placeholder="Backend team"
                  required
                  type="text"
                  value={newConversationForm.roomName}
                />
              </label>
            ) : (
              <label>
                User ID
                <input
                  onChange={(event) =>
                    updateNewConversationForm('otherUserID', event.target.value)
                  }
                  placeholder="UUID"
                  required
                  type="text"
                  value={newConversationForm.otherUserID}
                />
              </label>
            )}

            <button
              className="new-chat-button"
              disabled={isCreatingConversation}
              type="submit"
            >
              {isCreatingConversation ? 'Creating' : 'New chat'}
            </button>
          </form>

          {conversationError && (
            <p className="form-message error">{conversationError}</p>
          )}

          <div className="conversation-list">
            {isLoadingConversations ? (
              <div className="empty-list">Loading conversations</div>
            ) : conversations.length === 0 ? (
              <div className="empty-list">No conversations yet</div>
            ) : (
              conversations.map((conversation) => (
                <button
                  className="conversation-item"
                  data-active={conversation.id === selectedConversationID}
                  key={conversation.id}
                  onClick={() => setSelectedConversationID(conversation.id)}
                  type="button"
                >
                  <span>{conversationLabel(conversation)}</span>
                  <small>
                    {conversation.type === 'direct' ? 'DM' : 'Room'}
                    {conversation.unread_count ? ` · ${conversation.unread_count}` : ''}
                  </small>
                </button>
              ))
            )}
          </div>
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
            <p className="eyebrow">
              {selectedConversation ? selectedConversation.type : 'Ready'}
            </p>
            <h2>
              {selectedConversation
                ? conversationLabel(selectedConversation)
                : 'Create or choose a conversation.'}
            </h2>
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

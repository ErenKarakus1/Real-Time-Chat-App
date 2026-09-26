import { useEffect, useMemo, useState, type FormEvent } from 'react'
import './App.css'
import {
  ApiError,
  createDirect,
  createMessage,
  createRoom,
  conversationWebSocketURL,
  deleteMessage,
  listConversations,
  listMessages,
  login,
  me,
  register,
  updateMessage,
  type Conversation,
  type Message,
  type RealtimeEvent,
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
  const [messages, setMessages] = useState<Message[]>([])
  const [draftMessage, setDraftMessage] = useState('')
  const [isLoadingMessages, setIsLoadingMessages] = useState(false)
  const [isSendingMessage, setIsSendingMessage] = useState(false)
  const [messageError, setMessageError] = useState('')
  const [socketStatus, setSocketStatus] = useState<'idle' | 'connected' | 'offline'>(
    'idle',
  )
  const [editingMessageID, setEditingMessageID] = useState('')
  const [editingContent, setEditingContent] = useState('')
  const [pendingMessageID, setPendingMessageID] = useState('')

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

  useEffect(() => {
    if (!token || !selectedConversationID) {
      return
    }

    let isCurrent = true

    Promise.resolve()
      .then(() => {
        if (isCurrent) {
          setIsLoadingMessages(true)
          setMessageError('')
        }

        return listMessages(token, selectedConversationID)
      })
      .then((items) => {
        if (isCurrent) {
          setMessages(items)
        }
      })
      .catch((caughtError) => {
        if (isCurrent) {
          setMessageError(
            caughtError instanceof ApiError
              ? caughtError.message
              : 'Could not load messages.',
          )
        }
      })
      .finally(() => {
        if (isCurrent) {
          setIsLoadingMessages(false)
        }
      })

    return () => {
      isCurrent = false
    }
  }, [token, selectedConversationID])

  useEffect(() => {
    if (!token || !selectedConversationID) {
      return
    }

    const socket = new WebSocket(
      conversationWebSocketURL(token, selectedConversationID),
    )
    let isCurrent = true

    socket.addEventListener('open', () => {
      if (isCurrent) {
        setSocketStatus('connected')
      }
    })

    socket.addEventListener('message', (event) => {
      if (!isCurrent || typeof event.data !== 'string') {
        return
      }

      let realtimeEvent: RealtimeEvent
      try {
        realtimeEvent = JSON.parse(event.data) as RealtimeEvent
      } catch {
        return
      }
      if (realtimeEvent.type === 'message.created') {
        addMessage(realtimeEvent.data)
        return
      }

      if (realtimeEvent.type === 'message.updated') {
        setMessages((currentMessages) =>
          currentMessages.map((message) =>
            message.id === realtimeEvent.data.id ? realtimeEvent.data : message,
          ),
        )
        return
      }

      if (realtimeEvent.type === 'message.deleted') {
        setMessages((currentMessages) =>
          currentMessages.filter((message) => message.id !== realtimeEvent.data.id),
        )
      }
    })

    socket.addEventListener('close', () => {
      if (isCurrent) {
        setSocketStatus('offline')
      }
    })

    socket.addEventListener('error', () => {
      if (isCurrent) {
        setSocketStatus('offline')
      }
    })

    return () => {
      isCurrent = false
      socket.close()
    }
  }, [token, selectedConversationID])

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
    setMessages([])
    setDraftMessage('')
    setMessageError('')
    setSocketStatus('idle')
    setEditingMessageID('')
    setEditingContent('')
    setPendingMessageID('')
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

  function addMessage(message: Message) {
    setMessages((currentMessages) => {
      if (currentMessages.some((currentMessage) => currentMessage.id === message.id)) {
        return currentMessages
      }

      return [...currentMessages, message]
    })
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
      setMessages([])
      setSocketStatus('idle')
      setEditingMessageID('')
      setEditingContent('')
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

  async function handleSendMessage(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!token || !selectedConversationID) {
      return
    }

    const content = draftMessage.trim()
    if (!content) {
      return
    }

    setIsSendingMessage(true)
    setMessageError('')

    try {
      const message = await createMessage(token, selectedConversationID, { content })
      addMessage(message)
      setDraftMessage('')
    } catch (caughtError) {
      setMessageError(
        caughtError instanceof ApiError
          ? caughtError.message
          : 'Could not send message.',
      )
    } finally {
      setIsSendingMessage(false)
    }
  }

  function formatMessageTime(value: string) {
    return new Intl.DateTimeFormat(undefined, {
      hour: '2-digit',
      minute: '2-digit',
    }).format(new Date(value))
  }

  function startEditingMessage(message: Message) {
    setEditingMessageID(message.id)
    setEditingContent(message.content)
    setMessageError('')
  }

  function cancelEditingMessage() {
    setEditingMessageID('')
    setEditingContent('')
  }

  async function handleUpdateMessage(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!token || !selectedConversationID || !editingMessageID) {
      return
    }

    const content = editingContent.trim()
    if (!content) {
      return
    }

    setPendingMessageID(editingMessageID)
    setMessageError('')

    try {
      const message = await updateMessage(token, selectedConversationID, editingMessageID, {
        content,
      })
      setMessages((currentMessages) =>
        currentMessages.map((currentMessage) =>
          currentMessage.id === message.id ? message : currentMessage,
        ),
      )
      cancelEditingMessage()
    } catch (caughtError) {
      setMessageError(
        caughtError instanceof ApiError
          ? caughtError.message
          : 'Could not update message.',
      )
    } finally {
      setPendingMessageID('')
    }
  }

  async function handleDeleteMessage(messageID: string) {
    if (!token || !selectedConversationID) {
      return
    }

    setPendingMessageID(messageID)
    setMessageError('')

    try {
      await deleteMessage(token, selectedConversationID, messageID)
      setMessages((currentMessages) =>
        currentMessages.filter((message) => message.id !== messageID),
      )
      if (editingMessageID === messageID) {
        cancelEditingMessage()
      }
    } catch (caughtError) {
      setMessageError(
        caughtError instanceof ApiError
          ? caughtError.message
          : 'Could not delete message.',
      )
    } finally {
      setPendingMessageID('')
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
                  onClick={() => {
                    setMessages([])
                    setSocketStatus('idle')
                    setEditingMessageID('')
                    setEditingContent('')
                    setSelectedConversationID(conversation.id)
                  }}
                  type="button"
                >
                  <span>{conversationLabel(conversation)}</span>
                  <small>
                    {conversation.type === 'direct' ? 'DM' : 'Room'}
                    {conversation.unread_count ? ` - ${conversation.unread_count}` : ''}
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

          {selectedConversation ? (
            <div className="chat-panel">
              <div className="chat-title">
                <div className="chat-title-row">
                  <p className="eyebrow">{selectedConversation.type}</p>
                  <span className="socket-status" data-status={socketStatus}>
                    {socketStatus === 'connected' ? 'Live' : 'Offline'}
                  </span>
                </div>
                <h2>{conversationLabel(selectedConversation)}</h2>
              </div>

              <div className="message-list" aria-live="polite">
                {isLoadingMessages ? (
                  <div className="empty-chat">
                    <p className="eyebrow">Loading</p>
                    <h2>Messages are coming in.</h2>
                  </div>
                ) : messages.length === 0 ? (
                  <div className="empty-chat">
                    <p className="eyebrow">No messages</p>
                    <h2>Start the conversation.</h2>
                  </div>
                ) : (
                  messages.map((message) => {
                    const isOwnMessage = message.sender_id === user.id
                    const isEditingMessage = editingMessageID === message.id
                    const isPendingMessage = pendingMessageID === message.id

                    return (
                      <article
                        className="message-bubble"
                        data-own={isOwnMessage}
                        key={message.id}
                      >
                        {isEditingMessage ? (
                          <form className="edit-message-form" onSubmit={handleUpdateMessage}>
                            <input
                              aria-label="Edit message"
                              onChange={(event) => setEditingContent(event.target.value)}
                              type="text"
                              value={editingContent}
                            />
                            <div className="message-actions">
                              <button
                                disabled={isPendingMessage || !editingContent.trim()}
                                type="submit"
                              >
                                Save
                              </button>
                              <button type="button" onClick={cancelEditingMessage}>
                                Cancel
                              </button>
                            </div>
                          </form>
                        ) : (
                          <>
                            <p>{message.content}</p>
                            <div className="message-meta">
                              <time dateTime={message.created_at}>
                                {formatMessageTime(message.created_at)}
                              </time>
                              {message.updated_at !== message.created_at && (
                                <span>Edited</span>
                              )}
                            </div>
                            {isOwnMessage && (
                              <div className="message-actions">
                                <button
                                  disabled={isPendingMessage}
                                  type="button"
                                  onClick={() => startEditingMessage(message)}
                                >
                                  Edit
                                </button>
                                <button
                                  disabled={isPendingMessage}
                                  type="button"
                                  onClick={() => handleDeleteMessage(message.id)}
                                >
                                  Delete
                                </button>
                              </div>
                            )}
                          </>
                        )}
                      </article>
                    )
                  })
                )}
              </div>

              {messageError && <p className="form-message error">{messageError}</p>}

              <form className="composer" onSubmit={handleSendMessage}>
                <input
                  aria-label="Message"
                  onChange={(event) => setDraftMessage(event.target.value)}
                  placeholder="Write a message"
                  type="text"
                  value={draftMessage}
                />
                <button
                  className="primary-button"
                  disabled={isSendingMessage || !draftMessage.trim()}
                  type="submit"
                >
                  {isSendingMessage ? 'Sending' : 'Send'}
                </button>
              </form>
            </div>
          ) : (
            <div className="empty-chat">
              <p className="eyebrow">Ready</p>
              <h2>Create or choose a conversation.</h2>
            </div>
          )}
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

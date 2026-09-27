import { useEffect, useMemo, useRef, useState, type FormEvent } from 'react'
import './App.css'
import {
  addParticipant,
  ApiError,
  createDirect,
  createMessage,
  createRoom,
  conversationWebSocketURL,
  deleteMessage,
  deleteRoom,
  leaveRoom,
  listConversations,
  listMessages,
  listParticipants,
  login,
  markConversationRead,
  me,
  register,
  removeParticipant,
  searchUsers,
  updateMessage,
  updateRoom,
  type Conversation,
  type Message,
  type Participant,
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
  const [conversationSearch, setConversationSearch] = useState('')
  const [newConversationForm, setNewConversationForm] = useState(
    emptyNewConversationForm,
  )
  const [isCreatingConversation, setIsCreatingConversation] = useState(false)
  const [messages, setMessages] = useState<Message[]>([])
  const [draftMessage, setDraftMessage] = useState('')
  const [isLoadingMessages, setIsLoadingMessages] = useState(false)
  const [isSendingMessage, setIsSendingMessage] = useState(false)
  const [messageError, setMessageError] = useState('')
  const [socketError, setSocketError] = useState('')
  const [socketStatus, setSocketStatus] = useState<'idle' | 'connected' | 'offline'>(
    'idle',
  )
  const [editingMessageID, setEditingMessageID] = useState('')
  const [editingContent, setEditingContent] = useState('')
  const [pendingMessageID, setPendingMessageID] = useState('')
  const [typingUserIDs, setTypingUserIDs] = useState<string[]>([])
  const [roomNameDraft, setRoomNameDraft] = useState('')
  const [isUpdatingRoom, setIsUpdatingRoom] = useState(false)
  const [participants, setParticipants] = useState<Participant[]>([])
  const [participantUserID, setParticipantUserID] = useState('')
  const [participantSearch, setParticipantSearch] = useState('')
  const [participantSearchResults, setParticipantSearchResults] = useState<User[]>([])
  const [isLoadingParticipants, setIsLoadingParticipants] = useState(false)
  const [pendingParticipantID, setPendingParticipantID] = useState('')
  const [directSearch, setDirectSearch] = useState('')
  const [directSearchResults, setDirectSearchResults] = useState<User[]>([])
  const socketRef = useRef<WebSocket | null>(null)
  const messageListRef = useRef<HTMLDivElement | null>(null)
  const typingTimeoutRef = useRef<number | null>(null)

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

        return listConversations(token, conversationSearch)
      })
      .then((items) => {
        if (isCurrent) {
          setConversations(items)
          setSelectedConversationID((currentID) => {
            const nextID = currentID || items[0]?.id || ''
            const selectedConversation = items.find((item) => item.id === nextID)
            setRoomNameDraft(selectedConversation?.name ?? '')
            return nextID
          })
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
  }, [token, user, conversationSearch])

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
    messageListRef.current?.scrollTo({
      top: messageListRef.current.scrollHeight,
      behavior: 'smooth',
    })
  }, [messages, selectedConversationID])

  useEffect(() => {
    if (!token || directSearch.trim().length < 2) {
      return
    }

    let isCurrent = true

    searchUsers(token, directSearch)
      .then((results) => {
        if (isCurrent) {
          setDirectSearchResults(results.filter((result) => result.id !== user?.id))
        }
      })
      .catch(() => {
        if (isCurrent) {
          setDirectSearchResults([])
        }
      })

    return () => {
      isCurrent = false
    }
  }, [token, directSearch, user?.id])

  useEffect(() => {
    if (!token || participantSearch.trim().length < 2) {
      return
    }

    let isCurrent = true

    searchUsers(token, participantSearch)
      .then((results) => {
        if (isCurrent) {
          const currentParticipantIDs = new Set(
            participants.map((participant) => participant.user_id),
          )
          setParticipantSearchResults(
            results.filter(
              (result) => result.id !== user?.id && !currentParticipantIDs.has(result.id),
            ),
          )
        }
      })
      .catch(() => {
        if (isCurrent) {
          setParticipantSearchResults([])
        }
      })

    return () => {
      isCurrent = false
    }
  }, [token, participantSearch, participants, user?.id])

  const selectedConversation = useMemo(
    () =>
      conversations.find(
        (conversation) => conversation.id === selectedConversationID,
      ) ?? null,
    [conversations, selectedConversationID],
  )

  useEffect(() => {
    if (!token || !selectedConversation) {
      return
    }

    let isCurrent = true

    Promise.resolve()
      .then(() => {
        if (isCurrent) {
          setIsLoadingParticipants(true)
        }

        return listParticipants(token, selectedConversation.id)
      })
      .then((items) => {
        if (isCurrent) {
          setParticipants(items)
        }
      })
      .catch((caughtError) => {
        if (isCurrent) {
          setConversationError(
            caughtError instanceof ApiError
              ? caughtError.message
              : 'Could not load participants.',
          )
        }
      })
      .finally(() => {
        if (isCurrent) {
          setIsLoadingParticipants(false)
        }
      })

    return () => {
      isCurrent = false
    }
  }, [token, selectedConversation])

  useEffect(() => {
    if (!token || !selectedConversationID) {
      return
    }

    const socket = new WebSocket(
      conversationWebSocketURL(token, selectedConversationID),
    )
    socketRef.current = socket
    let isCurrent = true

    socket.addEventListener('open', () => {
      if (isCurrent) {
        setSocketError('')
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
        return
      }

      if (realtimeEvent.type === 'typing.started') {
        if (realtimeEvent.data.user_id !== user?.id) {
          setTypingUserIDs((currentUserIDs) =>
            currentUserIDs.includes(realtimeEvent.data.user_id)
              ? currentUserIDs
              : [...currentUserIDs, realtimeEvent.data.user_id],
          )
        }
        return
      }

      if (realtimeEvent.type === 'typing.stopped') {
        setTypingUserIDs((currentUserIDs) =>
          currentUserIDs.filter((userID) => userID !== realtimeEvent.data.user_id),
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
        setSocketError('Realtime connection failed. Messages still work after refresh.')
        setSocketStatus('offline')
      }
    })

    return () => {
      isCurrent = false
      socketRef.current = null
      socket.close()
    }
  }, [token, selectedConversationID, user?.id])

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
    setConversationSearch('')
    setMessages([])
    setDraftMessage('')
    setMessageError('')
    setSocketError('')
    setSocketStatus('idle')
    setEditingMessageID('')
    setEditingContent('')
    setPendingMessageID('')
    setTypingUserIDs([])
    setRoomNameDraft('')
    setParticipants([])
    setParticipantUserID('')
    setParticipantSearch('')
    setParticipantSearchResults([])
    setPendingParticipantID('')
    setDirectSearch('')
    setDirectSearchResults([])
    stopTyping()
  }

  function conversationLabel(conversation: Conversation) {
    if (conversation.name) {
      return conversation.name
    }

    return conversation.type === 'direct' ? 'Direct message' : 'Room'
  }

  function selectConversation(conversation: Conversation) {
    setMessages([])
    setSocketStatus('idle')
    setSocketError('')
    setEditingMessageID('')
    setEditingContent('')
    setTypingUserIDs([])
    setRoomNameDraft(conversation.name ?? '')
    setParticipants([])
    setParticipantUserID('')
    setParticipantSearch('')
    setParticipantSearchResults([])
    setPendingParticipantID('')
    stopTyping()
    setSelectedConversationID(conversation.id)
    if (token) {
      markConversationRead(token, conversation.id)
        .then(() => {
          setConversations((currentConversations) =>
            currentConversations.map((currentConversation) =>
              currentConversation.id === conversation.id
                ? { ...currentConversation, unread_count: 0 }
                : currentConversation,
            ),
          )
        })
        .catch(() => undefined)
    }
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

  function selectDirectUser(selectedUser: User) {
    updateNewConversationForm('otherUserID', selectedUser.id)
    setDirectSearch(selectedUser.username)
    setDirectSearchResults([])
  }

  function selectParticipantUser(selectedUser: User) {
    setParticipantUserID(selectedUser.id)
    setParticipantSearch(selectedUser.username)
    setParticipantSearchResults([])
  }

  function userOptionLabel(option: User) {
    return `${option.username} (${option.email})`
  }

  function participantLabel(participant: Participant) {
    if (participant.user_id === user?.id) {
      return 'You'
    }

    return participant.username || participant.email || participant.user_id
  }

  function typingLabel() {
    const names = typingUserIDs.map((userID) => {
      const participant = participants.find((item) => item.user_id === userID)
      return participant?.username || participant?.email || 'Someone'
    })

    if (names.length === 1) {
      return `${names[0]} is typing`
    }

    if (names.length === 2) {
      return `${names[0]} and ${names[1]} are typing`
    }

    return `${names[0]} and ${names.length - 1} others are typing`
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
      setSocketError('')
      setEditingMessageID('')
      setEditingContent('')
      setTypingUserIDs([])
      setRoomNameDraft(conversation.name ?? '')
      setParticipants([])
      setParticipantUserID('')
      setParticipantSearch('')
      setParticipantSearchResults([])
      setDirectSearch('')
      setDirectSearchResults([])
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
      stopTyping()
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

  function sendTypingEvent(type: 'typing.started' | 'typing.stopped') {
    const socket = socketRef.current
    if (!socket || socket.readyState !== WebSocket.OPEN) {
      return
    }

    socket.send(JSON.stringify({ type }))
  }

  function stopTyping() {
    if (typingTimeoutRef.current !== null) {
      window.clearTimeout(typingTimeoutRef.current)
      typingTimeoutRef.current = null
    }

    sendTypingEvent('typing.stopped')
  }

  function handleDraftMessageChange(value: string) {
    setDraftMessage(value)

    if (!value.trim()) {
      stopTyping()
      return
    }

    sendTypingEvent('typing.started')
    if (typingTimeoutRef.current !== null) {
      window.clearTimeout(typingTimeoutRef.current)
    }
    typingTimeoutRef.current = window.setTimeout(() => {
      stopTyping()
    }, 1600)
  }

  function formatMessageTime(value: string) {
    return new Intl.DateTimeFormat(undefined, {
      hour: '2-digit',
      minute: '2-digit',
    }).format(new Date(value))
  }

  function messageDeliveryLabel(message: Message) {
    if (message.sender_id !== user?.id) {
      return ''
    }

    const readByOtherParticipant = participants.some((participant) => {
      if (participant.user_id === user.id || !participant.last_read_at) {
        return false
      }

      return new Date(participant.last_read_at).getTime() >= new Date(message.created_at).getTime()
    })

    return readByOtherParticipant ? 'Read' : 'Sent'
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

  async function handleUpdateRoom(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!token || !selectedConversationID) {
      return
    }

    const name = roomNameDraft.trim()
    if (!name) {
      return
    }

    setIsUpdatingRoom(true)
    setConversationError('')

    try {
      const conversation = await updateRoom(token, selectedConversationID, { name })
      setConversations((currentConversations) =>
        currentConversations.map((currentConversation) =>
          currentConversation.id === conversation.id
            ? {
                ...currentConversation,
                ...conversation,
                unread_count: currentConversation.unread_count,
              }
            : currentConversation,
        ),
      )
      setRoomNameDraft(conversation.name ?? '')
    } catch (caughtError) {
      setConversationError(
        caughtError instanceof ApiError
          ? caughtError.message
          : 'Could not update room.',
      )
    } finally {
      setIsUpdatingRoom(false)
    }
  }

  async function handleDeleteRoom() {
    if (!token || !selectedConversationID) {
      return
    }

    setIsUpdatingRoom(true)
    setConversationError('')

    try {
      await deleteRoom(token, selectedConversationID)
      setConversations((currentConversations) =>
        currentConversations.filter(
          (conversation) => conversation.id !== selectedConversationID,
        ),
      )
      setSelectedConversationID('')
      setMessages([])
      setRoomNameDraft('')
      setSocketStatus('idle')
    } catch (caughtError) {
      setConversationError(
        caughtError instanceof ApiError
          ? caughtError.message
          : 'Could not delete room.',
      )
    } finally {
      setIsUpdatingRoom(false)
    }
  }

  async function handleAddParticipant(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!token || !selectedConversationID) {
      return
    }

    const userID = participantUserID.trim()
    if (!userID) {
      return
    }

    setPendingParticipantID(userID)
    setConversationError('')

    try {
      const participant = await addParticipant(token, selectedConversationID, {
        user_id: userID,
      })
      setParticipants((currentParticipants) => {
        if (
          currentParticipants.some(
            (currentParticipant) => currentParticipant.user_id === participant.user_id,
          )
        ) {
          return currentParticipants
        }

        return [...currentParticipants, participant]
      })
      setParticipantUserID('')
      setParticipantSearch('')
      setParticipantSearchResults([])
    } catch (caughtError) {
      setConversationError(
        caughtError instanceof ApiError
          ? caughtError.message
          : 'Could not add participant.',
      )
    } finally {
      setPendingParticipantID('')
    }
  }

  async function handleRemoveParticipant(userID: string) {
    if (!token || !selectedConversationID) {
      return
    }

    setPendingParticipantID(userID)
    setConversationError('')

    try {
      await removeParticipant(token, selectedConversationID, userID)
      setParticipants((currentParticipants) =>
        currentParticipants.filter((participant) => participant.user_id !== userID),
      )
    } catch (caughtError) {
      setConversationError(
        caughtError instanceof ApiError
          ? caughtError.message
          : 'Could not remove participant.',
      )
    } finally {
      setPendingParticipantID('')
    }
  }

  async function handleLeaveRoom() {
    if (!token || !selectedConversationID) {
      return
    }

    setIsUpdatingRoom(true)
    setConversationError('')

    try {
      await leaveRoom(token, selectedConversationID)
      setConversations((currentConversations) =>
        currentConversations.filter(
          (conversation) => conversation.id !== selectedConversationID,
        ),
      )
      setSelectedConversationID('')
      setMessages([])
      setParticipants([])
      setRoomNameDraft('')
      setSocketStatus('idle')
    } catch (caughtError) {
      setConversationError(
        caughtError instanceof ApiError ? caughtError.message : 'Could not leave room.',
      )
    } finally {
      setIsUpdatingRoom(false)
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
                User
                <input
                  onChange={(event) => {
                    const value = event.target.value
                    setDirectSearch(value)
                    if (value.trim().length < 2) {
                      setDirectSearchResults([])
                    }
                    updateNewConversationForm('otherUserID', '')
                  }}
                  placeholder="Search username or email"
                  required
                  type="search"
                  value={directSearch}
                />
                {directSearchResults.length > 0 && (
                  <div className="user-results">
                    {directSearchResults.map((result) => (
                      <button
                        key={result.id}
                        type="button"
                        onClick={() => selectDirectUser(result)}
                      >
                        {userOptionLabel(result)}
                      </button>
                    ))}
                  </div>
                )}
              </label>
            )}

            <button
              className="new-chat-button"
              disabled={
                isCreatingConversation ||
                (newConversationForm.mode === 'direct' &&
                  !newConversationForm.otherUserID)
              }
              type="submit"
            >
              {isCreatingConversation ? 'Creating' : 'New chat'}
            </button>
          </form>

          {conversationError && (
            <p className="form-message error">{conversationError}</p>
          )}

          <label className="conversation-search">
            Search
            <input
              onChange={(event) => setConversationSearch(event.target.value)}
              placeholder="Room name"
              type="search"
              value={conversationSearch}
            />
          </label>

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
                  data-unread={Boolean(conversation.unread_count)}
                  key={conversation.id}
                  onClick={() => selectConversation(conversation)}
                  type="button"
                >
                  <span className="conversation-name">
                    {conversationLabel(conversation)}
                  </span>
                  <span className="conversation-meta">
                    <small>{conversation.type === 'direct' ? 'DM' : 'Room'}</small>
                    {Boolean(conversation.unread_count) && (
                      <strong aria-label={`${conversation.unread_count} unread messages`}>
                        {conversation.unread_count}
                      </strong>
                    )}
                  </span>
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
                {selectedConversation.type === 'room' && (
                  <div className="room-panel">
                    <form className="room-tools" onSubmit={handleUpdateRoom}>
                      <input
                        aria-label="Room name"
                        minLength={2}
                        onChange={(event) => setRoomNameDraft(event.target.value)}
                        type="text"
                        value={roomNameDraft}
                      />
                      <button
                        disabled={
                          isUpdatingRoom ||
                          !roomNameDraft.trim() ||
                          roomNameDraft.trim() === selectedConversation.name
                        }
                        type="submit"
                      >
                        Rename
                      </button>
                      <button
                        disabled={isUpdatingRoom}
                        type="button"
                        onClick={handleDeleteRoom}
                      >
                        Delete
                      </button>
                      <button
                        disabled={isUpdatingRoom}
                        type="button"
                        onClick={handleLeaveRoom}
                      >
                        Leave
                      </button>
                    </form>

                    <form className="participant-form" onSubmit={handleAddParticipant}>
                      <input
                        aria-label="Participant"
                        onChange={(event) => {
                          const value = event.target.value
                          setParticipantSearch(value)
                          if (value.trim().length < 2) {
                            setParticipantSearchResults([])
                          }
                          setParticipantUserID('')
                        }}
                        placeholder="Search username or email"
                        type="search"
                        value={participantSearch}
                      />
                      <button
                        disabled={
                          !participantUserID ||
                          pendingParticipantID === participantUserID
                        }
                        type="submit"
                      >
                        Add
                      </button>
                    </form>

                    {participantSearchResults.length > 0 && (
                      <div className="user-results">
                        {participantSearchResults.map((result) => (
                          <button
                            key={result.id}
                            type="button"
                            onClick={() => selectParticipantUser(result)}
                          >
                            {userOptionLabel(result)}
                          </button>
                        ))}
                      </div>
                    )}

                    <div className="participant-list">
                      {isLoadingParticipants ? (
                        <span>Loading members</span>
                      ) : participants.length === 0 ? (
                        <span>No members loaded</span>
                      ) : (
                        participants.map((participant) => (
                          <div className="participant-row" key={participant.user_id}>
                            <span>
                              {participantLabel(participant)}
                            </span>
                            <strong>{participant.role}</strong>
                            {participant.user_id !== user.id && (
                              <button
                                disabled={pendingParticipantID === participant.user_id}
                                type="button"
                                onClick={() =>
                                  handleRemoveParticipant(participant.user_id)
                                }
                              >
                                Remove
                              </button>
                            )}
                          </div>
                        ))
                      )}
                    </div>
                  </div>
                )}
              </div>

              <div className="message-list" aria-live="polite" ref={messageListRef}>
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
                    const deliveryLabel = messageDeliveryLabel(message)

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
                              {deliveryLabel && (
                                <span className="delivery-status">{deliveryLabel}</span>
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
              {socketError && <p className="form-message warning">{socketError}</p>}
              {typingUserIDs.length > 0 && (
                <p className="typing-indicator">{typingLabel()}</p>
              )}

              <form className="composer" onSubmit={handleSendMessage}>
                <input
                  aria-label="Message"
                  onChange={(event) => handleDraftMessageChange(event.target.value)}
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

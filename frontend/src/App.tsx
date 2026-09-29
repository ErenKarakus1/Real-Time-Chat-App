import {
  Fragment,
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
  type FormEvent,
} from 'react'
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
  getPresence,
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
  transferOwnership,
  updateMessage,
  updateParticipantRole,
  updateRoom,
  type Conversation,
  type Message,
  type Participant,
  type RealtimeEvent,
  type User,
} from './api'

const TOKEN_KEY = 'rtc_token'
const MESSAGE_PAGE_SIZE = 30

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
  const [messagesConversationID, setMessagesConversationID] = useState('')
  const [unreadDivider, setUnreadDivider] = useState<{
    conversationID: string
    lastReadAt: string | null
  } | null>(null)
  const [draftMessage, setDraftMessage] = useState('')
  const [isLoadingMessages, setIsLoadingMessages] = useState(false)
  const [isLoadingOlderMessages, setIsLoadingOlderMessages] = useState(false)
  const [hasOlderMessages, setHasOlderMessages] = useState(false)
  const [isAtMessageEnd, setIsAtMessageEnd] = useState(true)
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
  const [isRoomSettingsOpen, setIsRoomSettingsOpen] = useState(false)
  const [isUpdatingRoom, setIsUpdatingRoom] = useState(false)
  const [participants, setParticipants] = useState<Participant[]>([])
  const [participantsByConversation, setParticipantsByConversation] = useState<
    Record<string, Participant[]>
  >({})
  const [presenceByUserID, setPresenceByUserID] = useState<Record<string, boolean>>({})
  const [participantUserID, setParticipantUserID] = useState('')
  const [participantSearch, setParticipantSearch] = useState('')
  const [participantSearchResults, setParticipantSearchResults] = useState<User[]>([])
  const [isLoadingParticipants, setIsLoadingParticipants] = useState(false)
  const [pendingParticipantID, setPendingParticipantID] = useState('')
  const [directSearch, setDirectSearch] = useState('')
  const [directSearchResults, setDirectSearchResults] = useState<User[]>([])
  const socketRef = useRef<WebSocket | null>(null)
  const conversationSocketsRef = useRef<Map<string, WebSocket>>(new Map())
  const selectedConversationIDRef = useRef(selectedConversationID)
  const tokenRef = useRef(token)
  const userIDRef = useRef(user?.id ?? '')
  const messageListRef = useRef<HTMLDivElement | null>(null)
  const unreadDividerRef = useRef<HTMLDivElement | null>(null)
  const messageEndRef = useRef<HTMLDivElement | null>(null)
  const shouldScrollToBottomRef = useRef(true)
  const forceScrollToBottomRef = useRef(false)
  const isAtMessageEndRef = useRef(true)
  const restoreMessageScrollRef = useRef<{
    scrollHeight: number
    scrollTop: number
  } | null>(null)
  const typingTimeoutRef = useRef<number | null>(null)

  useEffect(() => {
    selectedConversationIDRef.current = selectedConversationID
  }, [selectedConversationID])

  useEffect(() => {
    tokenRef.current = token
  }, [token])

  useEffect(() => {
    userIDRef.current = user?.id ?? ''
  }, [user?.id])

  useEffect(() => {
    isAtMessageEndRef.current = isAtMessageEnd
  }, [isAtMessageEnd])

  const unreadDividerMessageID = useMemo(() => {
    if (
      !user ||
      !unreadDivider ||
      unreadDivider.conversationID !== selectedConversationID ||
      messagesConversationID !== selectedConversationID
    ) {
      return ''
    }

    return (
      messages.find((message) => {
        if (message.sender_id === user.id) {
          return false
        }

        return (
          !unreadDivider.lastReadAt ||
          new Date(message.created_at).getTime() >
            new Date(unreadDivider.lastReadAt).getTime()
        )
      })?.id ?? ''
    )
  }, [messages, messagesConversationID, selectedConversationID, unreadDivider, user])

  useEffect(() => {
    const conversationSockets = conversationSocketsRef.current

    return () => {
      conversationSockets.forEach((socket) => socket.close())
      conversationSockets.clear()
      socketRef.current = null
    }
  }, [])

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
          setHasOlderMessages(false)
          setIsAtMessageEnd(true)
          shouldScrollToBottomRef.current = true
        }

        return listMessages(token, selectedConversationID, {
          limit: MESSAGE_PAGE_SIZE,
        })
      })
      .then((items) => {
        if (isCurrent) {
          setMessages(items)
          setMessagesConversationID(selectedConversationID)
          setHasOlderMessages(items.length === MESSAGE_PAGE_SIZE)
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

  useLayoutEffect(() => {
    const messageList = messageListRef.current
    if (
      !messageList ||
      messages.length === 0 ||
      messagesConversationID !== selectedConversationID
    ) {
      return
    }

    const restoreScroll = restoreMessageScrollRef.current
    if (restoreScroll) {
      messageList.scrollTop =
        messageList.scrollHeight - restoreScroll.scrollHeight + restoreScroll.scrollTop
      restoreMessageScrollRef.current = null
      return
    }

    if (shouldScrollToBottomRef.current) {
      const shouldScrollToUnreadDivider =
        Boolean(unreadDividerMessageID) && !forceScrollToBottomRef.current

      let secondFrame = 0
      const firstFrame = requestAnimationFrame(() => {
        secondFrame = requestAnimationFrame(() => {
          if (!shouldScrollToUnreadDivider) {
            messageList.scrollTop = messageList.scrollHeight

            if (forceScrollToBottomRef.current) {
              requestAnimationFrame(() => {
                messageList.scrollTop = messageList.scrollHeight
              })
            }
          } else {
            const target = unreadDividerRef.current

            if (!target) {
              return
            }

            target.scrollIntoView({
              behavior: 'auto',
              block: 'start',
            })
          }

          const isAtEnd =
            messageList.scrollHeight - messageList.scrollTop - messageList.clientHeight < 48
          setIsAtMessageEnd(isAtEnd)
          isAtMessageEndRef.current = isAtEnd
          forceScrollToBottomRef.current = false
          shouldScrollToBottomRef.current = false
        })
      })

      return () => {
        cancelAnimationFrame(firstFrame)
        cancelAnimationFrame(secondFrame)
      }
    }
  }, [
    messages,
    messagesConversationID,
    selectedConversationID,
    unreadDividerMessageID,
  ])

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

  useEffect(() => {
    if (!token || !user || conversations.length === 0) {
      return
    }

    let isCurrent = true
    const missingConversations = conversations.filter(
      (conversation) => !participantsByConversation[conversation.id],
    )

    if (missingConversations.length === 0) {
      return
    }

    Promise.all(
      missingConversations.map((conversation) =>
        listParticipants(token, conversation.id)
          .then((items) => [conversation.id, items] as const)
          .catch(() => [conversation.id, [] as Participant[]] as const),
      ),
    ).then((results) => {
      if (!isCurrent) {
        return
      }

      setParticipantsByConversation((currentParticipants) => {
        const nextParticipants = { ...currentParticipants }
        results.forEach(([conversationID, items]) => {
          nextParticipants[conversationID] = items
        })
        return nextParticipants
      })
    })

    return () => {
      isCurrent = false
    }
  }, [token, user, conversations, participantsByConversation])

  useEffect(() => {
    if (!token) {
      return
    }

    const userIDs = new Set<string>()
    Object.values(participantsByConversation).forEach((conversationParticipants) => {
      conversationParticipants.forEach((participant) => userIDs.add(participant.user_id))
    })
    directSearchResults.forEach((result) => userIDs.add(result.id))
    participantSearchResults.forEach((result) => userIDs.add(result.id))

    if (userIDs.size === 0) {
      return
    }

    let isCurrent = true
    getPresence(token, [...userIDs])
      .then((statuses) => {
        if (!isCurrent) {
          return
        }

        setPresenceByUserID((currentPresence) => {
          const nextPresence = { ...currentPresence }
          statuses.forEach((status) => {
            nextPresence[status.user_id] = status.online
          })
          return nextPresence
        })
      })
      .catch(() => undefined)

    return () => {
      isCurrent = false
    }
  }, [token, participantsByConversation, directSearchResults, participantSearchResults])

  const selectedConversation = useMemo(
    () =>
      conversations.find(
        (conversation) => conversation.id === selectedConversationID,
      ) ?? null,
    [conversations, selectedConversationID],
  )

  const currentParticipant = useMemo(
    () => participants.find((participant) => participant.user_id === user?.id) ?? null,
    [participants, user?.id],
  )

  const canManageRoom = currentParticipant?.role === 'owner'
  const canManageMembers =
    currentParticipant?.role === 'owner' || currentParticipant?.role === 'admin'
  const canLeaveRoom = Boolean(currentParticipant && currentParticipant.role !== 'owner')

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
          setUnreadDivider((currentDivider) => {
            if (currentDivider?.conversationID === selectedConversation.id) {
              return currentDivider
            }

            const currentUserParticipant = items.find(
              (participant) => participant.user_id === user?.id,
            )
            if (currentUserParticipant?.last_read_at) {
              shouldScrollToBottomRef.current = true
            }
            return {
              conversationID: selectedConversation.id,
              lastReadAt: currentUserParticipant?.last_read_at ?? null,
            }
          })
          setParticipantsByConversation((currentParticipants) => ({
            ...currentParticipants,
            [selectedConversation.id]: items,
          }))
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
  }, [token, selectedConversation, user?.id])

  const handleRealtimeEvent = useCallback(
    (conversationID: string, realtimeEvent: RealtimeEvent) => {
      const currentSelectedConversationID = selectedConversationIDRef.current
      const currentUserID = userIDRef.current
      const currentToken = tokenRef.current

      if (realtimeEvent.type === 'message.created') {
        if (conversationID === currentSelectedConversationID) {
          setMessagesConversationID(conversationID)
          setMessages((currentMessages) => {
            if (
              currentMessages.some(
                (currentMessage) => currentMessage.id === realtimeEvent.data.id,
              )
            ) {
              return currentMessages
            }

            return [...currentMessages, realtimeEvent.data]
          })

          if (realtimeEvent.data.sender_id !== currentUserID && currentToken) {
            markConversationRead(currentToken, conversationID).catch(() => undefined)
          }
        }

        setConversations((currentConversations) =>
          currentConversations.map((currentConversation) => {
            if (currentConversation.id !== conversationID) {
              return currentConversation
            }

            const shouldIncrementUnread =
              conversationID !== currentSelectedConversationID &&
              realtimeEvent.data.sender_id !== currentUserID

            return {
              ...currentConversation,
              unread_count: shouldIncrementUnread
                ? (currentConversation.unread_count ?? 0) + 1
                : currentConversation.unread_count,
            }
          }),
        )
        return
      }

      if (realtimeEvent.type === 'message.updated') {
        if (conversationID === currentSelectedConversationID) {
          setMessages((currentMessages) =>
            currentMessages.map((message) =>
              message.id === realtimeEvent.data.id ? realtimeEvent.data : message,
            ),
          )
        }
        return
      }

      if (realtimeEvent.type === 'message.deleted') {
        if (conversationID === currentSelectedConversationID) {
          setMessages((currentMessages) =>
            currentMessages.filter((message) => message.id !== realtimeEvent.data.id),
          )
        }
        return
      }

      if (realtimeEvent.type === 'conversation.read') {
        if (realtimeEvent.data.user_id === currentUserID) {
          setConversations((currentConversations) =>
            currentConversations.map((currentConversation) =>
              currentConversation.id === realtimeEvent.data.conversation_id
                ? { ...currentConversation, unread_count: 0 }
                : currentConversation,
            ),
          )
        }

        setParticipantsByConversation((currentParticipants) => {
          const conversationParticipants = currentParticipants[conversationID]
          if (!conversationParticipants) {
            return currentParticipants
          }

          return {
            ...currentParticipants,
            [conversationID]: conversationParticipants.map((participant) =>
              participant.user_id === realtimeEvent.data.user_id
                ? { ...participant, last_read_at: realtimeEvent.data.last_read_at }
                : participant,
            ),
          }
        })

        if (conversationID === currentSelectedConversationID) {
          setParticipants((currentParticipants) =>
            currentParticipants.map((participant) =>
              participant.user_id === realtimeEvent.data.user_id
                ? { ...participant, last_read_at: realtimeEvent.data.last_read_at }
                : participant,
            ),
          )
        }
        return
      }

      if (conversationID !== currentSelectedConversationID) {
        return
      }

      if (realtimeEvent.type === 'typing.started') {
        if (realtimeEvent.data.user_id !== currentUserID) {
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
    },
    [],
  )

  useEffect(() => {
    if (!token || !user) {
      conversationSocketsRef.current.forEach((socket) => socket.close())
      conversationSocketsRef.current.clear()
      socketRef.current = null
      return
    }

    const conversationIDs = new Set(conversations.map((conversation) => conversation.id))

    conversationSocketsRef.current.forEach((socket, conversationID) => {
      if (!conversationIDs.has(conversationID)) {
        socket.close()
        conversationSocketsRef.current.delete(conversationID)
      }
    })

    conversations.forEach((conversation) => {
      if (conversationSocketsRef.current.has(conversation.id)) {
        return
      }

      const socket = new WebSocket(conversationWebSocketURL(token, conversation.id))
      conversationSocketsRef.current.set(conversation.id, socket)

      socket.addEventListener('open', () => {
        if (conversation.id === selectedConversationIDRef.current) {
          setSocketError('')
          setSocketStatus('connected')
        }
      })

      socket.addEventListener('message', (event) => {
        if (typeof event.data !== 'string') {
          return
        }

        let realtimeEvent: RealtimeEvent
        try {
          realtimeEvent = JSON.parse(event.data) as RealtimeEvent
        } catch {
          return
        }

        handleRealtimeEvent(conversation.id, realtimeEvent)
      })

      socket.addEventListener('close', () => {
        conversationSocketsRef.current.delete(conversation.id)
        if (conversation.id === selectedConversationIDRef.current) {
          setSocketStatus('offline')
        }
      })

      socket.addEventListener('error', () => {
        if (conversation.id === selectedConversationIDRef.current) {
          setSocketError('Realtime connection failed. Messages still work after refresh.')
          setSocketStatus('offline')
        }
      })
    })

    socketRef.current = conversationSocketsRef.current.get(selectedConversationID) ?? null

  }, [token, user, conversations, selectedConversationID, handleRealtimeEvent])

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
    setMessagesConversationID('')
    setUnreadDivider(null)
    setHasOlderMessages(false)
    setIsLoadingOlderMessages(false)
    setIsAtMessageEnd(true)
    setDraftMessage('')
    setMessageError('')
    setSocketError('')
    setSocketStatus('idle')
    setEditingMessageID('')
    setEditingContent('')
    setPendingMessageID('')
    setTypingUserIDs([])
    setRoomNameDraft('')
    setIsRoomSettingsOpen(false)
    setParticipants([])
    setParticipantsByConversation({})
    setPresenceByUserID({})
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

    if (conversation.type === 'direct') {
      const otherParticipant = participantsByConversation[conversation.id]?.find(
        (participant) => participant.user_id !== user?.id,
      )
      return otherParticipant
        ? participantLabel(otherParticipant)
        : 'Direct message'
    }

    return 'Room'
  }

  function conversationPresenceLabel(conversation: Conversation) {
    if (conversation.type !== 'direct') {
      return ''
    }

    const otherParticipant = participantsByConversation[conversation.id]?.find(
      (participant) => participant.user_id !== user?.id,
    )
    if (!otherParticipant) {
      return ''
    }

    return presenceByUserID[otherParticipant.user_id] ? 'Online' : 'Offline'
  }

  function selectConversation(conversation: Conversation) {
    if (conversation.id === selectedConversationID) {
      return
    }

    setMessages([])
    setMessagesConversationID('')
    const cachedCurrentParticipant = participantsByConversation[conversation.id]?.find(
      (participant) => participant.user_id === user?.id,
    )
    setUnreadDivider({
      conversationID: conversation.id,
      lastReadAt: cachedCurrentParticipant?.last_read_at ?? null,
    })
    setHasOlderMessages(false)
    setIsLoadingOlderMessages(false)
    setIsAtMessageEnd(true)
    shouldScrollToBottomRef.current = true
    restoreMessageScrollRef.current = null
    const existingSocket = conversationSocketsRef.current.get(conversation.id)
    setSocketStatus(
      existingSocket?.readyState === WebSocket.OPEN ? 'connected' : 'idle',
    )
    setSocketError('')
    setEditingMessageID('')
    setEditingContent('')
    setTypingUserIDs([])
    setRoomNameDraft(conversation.name ?? '')
    setIsRoomSettingsOpen(false)
    setParticipants([])
    setParticipantUserID('')
    setParticipantSearch('')
    setParticipantSearchResults([])
    setPendingParticipantID('')
    stopTyping()
    setSelectedConversationID(conversation.id)
    if (token) {
      const markRead = () =>
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

      listParticipants(token, conversation.id)
        .then((items) => {
          const currentUserParticipant = items.find(
            (participant) => participant.user_id === user?.id,
          )
          if (currentUserParticipant?.last_read_at) {
            shouldScrollToBottomRef.current = true
          }
          setUnreadDivider({
            conversationID: conversation.id,
            lastReadAt: currentUserParticipant?.last_read_at ?? null,
          })
          setParticipantsByConversation((currentParticipants) => ({
            ...currentParticipants,
            [conversation.id]: items,
          }))
          if (conversation.id === selectedConversationIDRef.current) {
            setParticipants(items)
          }
          return markRead()
        })
        .catch(() => {
          void markRead()
        })
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

  function presenceLabel(userID: string) {
    return presenceByUserID[userID] ? 'Online' : 'Offline'
  }

  function participantLabel(participant: Participant) {
    if (participant.user_id === user?.id) {
      return 'You'
    }

    return participant.username || participant.email || participant.user_id
  }

  function mergeParticipantIdentity(
    currentParticipant: Participant,
    updatedParticipant: Participant,
  ) {
    return {
      ...currentParticipant,
      ...updatedParticipant,
      username: updatedParticipant.username || currentParticipant.username,
      email: updatedParticipant.email || currentParticipant.email,
    }
  }

  function canRemoveParticipant(participant: Participant) {
    if (!currentParticipant || participant.user_id === user?.id) {
      return false
    }

    if (currentParticipant.role === 'owner') {
      return participant.role !== 'owner'
    }

    return currentParticipant.role === 'admin' && participant.role === 'member'
  }

  function canChangeParticipantRole(participant: Participant) {
    return (
      currentParticipant?.role === 'owner' &&
      participant.user_id !== user?.id &&
      participant.role !== 'owner'
    )
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

  function addMessage(message: Message, options: { forceBottom?: boolean } = {}) {
    if (options.forceBottom) {
      shouldScrollToBottomRef.current = true
      forceScrollToBottomRef.current = true
    } else {
      shouldScrollToBottomRef.current = isAtMessageEndRef.current
    }

    setMessages((currentMessages) => {
      if (currentMessages.some((currentMessage) => currentMessage.id === message.id)) {
        return currentMessages
      }

      return [...currentMessages, message]
    })
  }

  async function loadOlderMessages() {
    if (
      !token ||
      !selectedConversationID ||
      !hasOlderMessages ||
      isLoadingOlderMessages ||
      messages.length === 0
    ) {
      return
    }

    const messageList = messageListRef.current
    if (messageList) {
      restoreMessageScrollRef.current = {
        scrollHeight: messageList.scrollHeight,
        scrollTop: messageList.scrollTop,
      }
    }
    shouldScrollToBottomRef.current = false
    setIsLoadingOlderMessages(true)
    setMessageError('')

    try {
      const olderMessages = await listMessages(token, selectedConversationID, {
        before: messages[0].created_at,
        limit: MESSAGE_PAGE_SIZE,
      })
      setHasOlderMessages(olderMessages.length === MESSAGE_PAGE_SIZE)
      setMessages((currentMessages) => {
        const currentMessageIDs = new Set(
          currentMessages.map((message) => message.id),
        )
        return [
          ...olderMessages.filter((message) => !currentMessageIDs.has(message.id)),
          ...currentMessages,
        ]
      })
    } catch (caughtError) {
      restoreMessageScrollRef.current = null
      setMessageError(
        caughtError instanceof ApiError
          ? caughtError.message
          : 'Could not load older messages.',
      )
    } finally {
      setIsLoadingOlderMessages(false)
    }
  }

  function handleMessageListScroll() {
    const messageList = messageListRef.current
    if (!messageList) {
      return
    }

    setIsAtMessageEnd(
      messageList.scrollHeight - messageList.scrollTop - messageList.clientHeight < 48,
    )

    if (messageList.scrollTop <= 24) {
      void loadOlderMessages()
    }
  }

  function scrollToMessageEnd() {
    messageListRef.current?.scrollTo({
      top: messageListRef.current.scrollHeight,
      behavior: 'smooth',
    })
    setIsAtMessageEnd(true)
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
      setMessagesConversationID('')
      setUnreadDivider(null)
      setHasOlderMessages(false)
      setIsLoadingOlderMessages(false)
      setIsAtMessageEnd(true)
      shouldScrollToBottomRef.current = true
      restoreMessageScrollRef.current = null
      setSocketStatus('idle')
      setSocketError('')
      setEditingMessageID('')
      setEditingContent('')
      setTypingUserIDs([])
      setRoomNameDraft(conversation.name ?? '')
      setIsRoomSettingsOpen(false)
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
      addMessage(message, { forceBottom: true })
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

  function messageDeliveryStatus(message: Message): 'sent' | 'read' | '' {
    if (message.sender_id !== user?.id) {
      return ''
    }

    const readByOtherParticipant = participants.some((participant) => {
      if (participant.user_id === user.id || !participant.last_read_at) {
        return false
      }

      return new Date(participant.last_read_at).getTime() >= new Date(message.created_at).getTime()
    })

    return readByOtherParticipant ? 'read' : 'sent'
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
      setMessagesConversationID('')
      setUnreadDivider(null)
      setHasOlderMessages(false)
      setIsLoadingOlderMessages(false)
      setIsAtMessageEnd(true)
      setRoomNameDraft('')
      setParticipantsByConversation((currentParticipants) => {
        const nextParticipants = { ...currentParticipants }
        delete nextParticipants[selectedConversationID]
        return nextParticipants
      })
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
      setParticipantsByConversation((currentParticipants) => {
        const conversationParticipants =
          currentParticipants[selectedConversationID] ?? []
        if (
          conversationParticipants.some(
            (currentParticipant) => currentParticipant.user_id === participant.user_id,
          )
        ) {
          return currentParticipants
        }

        return {
          ...currentParticipants,
          [selectedConversationID]: [...conversationParticipants, participant],
        }
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
      setParticipantsByConversation((currentParticipants) => ({
        ...currentParticipants,
        [selectedConversationID]: (
          currentParticipants[selectedConversationID] ?? []
        ).filter((participant) => participant.user_id !== userID),
      }))
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

  async function handleUpdateParticipantRole(
    participant: Participant,
    role: 'admin' | 'member',
  ) {
    if (!token || !selectedConversationID) {
      return
    }

    setPendingParticipantID(participant.user_id)
    setConversationError('')

    try {
      const updatedParticipant = await updateParticipantRole(
        token,
        selectedConversationID,
        participant.user_id,
        { role },
      )
      setParticipants((currentParticipants) =>
        currentParticipants.map((currentParticipant) =>
          currentParticipant.user_id === updatedParticipant.user_id
            ? mergeParticipantIdentity(currentParticipant, updatedParticipant)
            : currentParticipant,
        ),
      )
      setParticipantsByConversation((currentParticipants) => ({
        ...currentParticipants,
        [selectedConversationID]: (currentParticipants[selectedConversationID] ?? []).map(
          (currentParticipant) =>
            currentParticipant.user_id === updatedParticipant.user_id
              ? mergeParticipantIdentity(currentParticipant, updatedParticipant)
              : currentParticipant,
        ),
      }))
    } catch (caughtError) {
      setConversationError(
        caughtError instanceof ApiError
          ? caughtError.message
          : 'Could not update participant role.',
      )
    } finally {
      setPendingParticipantID('')
    }
  }

  async function handleTransferOwnership(participant: Participant) {
    if (!token || !selectedConversationID) {
      return
    }

    setPendingParticipantID(participant.user_id)
    setConversationError('')

    try {
      const newOwner = await transferOwnership(token, selectedConversationID, {
        user_id: participant.user_id,
      })
      setParticipants((currentParticipants) =>
        currentParticipants.map((currentParticipant) => {
          if (currentParticipant.user_id === user?.id) {
            return { ...currentParticipant, role: 'admin' }
          }

          return currentParticipant.user_id === newOwner.user_id
            ? mergeParticipantIdentity(currentParticipant, newOwner)
            : currentParticipant
        }),
      )
      setParticipantsByConversation((currentParticipants) => ({
        ...currentParticipants,
        [selectedConversationID]: (currentParticipants[selectedConversationID] ?? []).map(
          (currentParticipant) => {
            if (currentParticipant.user_id === user?.id) {
              return { ...currentParticipant, role: 'admin' }
            }

            return currentParticipant.user_id === newOwner.user_id
              ? mergeParticipantIdentity(currentParticipant, newOwner)
              : currentParticipant
          },
        ),
      }))
    } catch (caughtError) {
      setConversationError(
        caughtError instanceof ApiError
          ? caughtError.message
          : 'Could not transfer ownership.',
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
      setMessagesConversationID('')
      setUnreadDivider(null)
      setHasOlderMessages(false)
      setIsLoadingOlderMessages(false)
      setIsAtMessageEnd(true)
      setParticipants([])
      setRoomNameDraft('')
      setParticipantsByConversation((currentParticipants) => {
        const nextParticipants = { ...currentParticipants }
        delete nextParticipants[selectedConversationID]
        return nextParticipants
      })
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
                        <span>{userOptionLabel(result)}</span>
                        <small>{presenceLabel(result.id)}</small>
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
                    <small>
                      {conversation.type === 'direct'
                        ? `DM${conversationPresenceLabel(conversation) ? ` - ${conversationPresenceLabel(conversation)}` : ''}`
                        : 'Room'}
                    </small>
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
                  <div className="chat-title-actions">
                    {selectedConversation.type === 'room' && (
                      <button
                        className="icon-button"
                        type="button"
                        aria-label="Room settings"
                        title="Room settings"
                        aria-expanded={isRoomSettingsOpen}
                        onClick={() =>
                          setIsRoomSettingsOpen((currentValue) => !currentValue)
                        }
                      >
                        ⚙
                      </button>
                    )}
                    <span className="socket-status" data-status={socketStatus}>
                      {socketStatus === 'connected' ? 'Live' : 'Offline'}
                    </span>
                  </div>
                </div>
                <h2>{conversationLabel(selectedConversation)}</h2>
                {selectedConversation.type === 'direct' &&
                  conversationPresenceLabel(selectedConversation) && (
                    <p className="chat-subtitle">
                      {conversationPresenceLabel(selectedConversation)}
                    </p>
                  )}
                {selectedConversation.type === 'room' && isRoomSettingsOpen && (
                  <div className="room-panel">
                    <div className="room-tools-stack">
                      {canManageRoom && (
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
                        </form>
                      )}

                      {canLeaveRoom && (
                        <button
                          className="room-action-button"
                          disabled={isUpdatingRoom}
                          type="button"
                          onClick={handleLeaveRoom}
                        >
                          Leave
                        </button>
                      )}
                    </div>

                    {canManageMembers && (
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
                    )}

                    {canManageMembers && participantSearchResults.length > 0 && (
                      <div className="user-results">
                        {participantSearchResults.map((result) => (
                          <button
                            key={result.id}
                            type="button"
                            onClick={() => selectParticipantUser(result)}
                          >
                            <span>{userOptionLabel(result)}</span>
                            <small>{presenceLabel(result.id)}</small>
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
                              {participant.user_id !== user.id && (
                                <small>{presenceLabel(participant.user_id)}</small>
                              )}
                            </span>
                            <strong>{participant.role}</strong>
                            {(canChangeParticipantRole(participant) ||
                              canRemoveParticipant(participant)) && (
                              <div className="participant-actions">
                                {canChangeParticipantRole(participant) && (
                                  <>
                                    {participant.role === 'member' ? (
                                      <button
                                        disabled={
                                          pendingParticipantID === participant.user_id
                                        }
                                        type="button"
                                        onClick={() =>
                                          handleUpdateParticipantRole(
                                            participant,
                                            'admin',
                                          )
                                        }
                                      >
                                        Make admin
                                      </button>
                                    ) : (
                                      <button
                                        disabled={
                                          pendingParticipantID === participant.user_id
                                        }
                                        type="button"
                                        onClick={() =>
                                          handleUpdateParticipantRole(
                                            participant,
                                            'member',
                                          )
                                        }
                                      >
                                        Make member
                                      </button>
                                    )}
                                    <button
                                      disabled={
                                        pendingParticipantID === participant.user_id
                                      }
                                      type="button"
                                      onClick={() => handleTransferOwnership(participant)}
                                    >
                                      Make owner
                                    </button>
                                  </>
                                )}
                                {canRemoveParticipant(participant) && (
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
                            )}
                          </div>
                        ))
                      )}
                    </div>
                  </div>
                )}
              </div>

              <div
                className="message-list"
                aria-live="polite"
                ref={messageListRef}
                onScroll={handleMessageListScroll}
              >
                {isLoadingMessages ||
                messagesConversationID !== selectedConversationID ? (
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
                  <>
                    {isLoadingOlderMessages && (
                      <div className="older-messages-status">
                        Loading older messages
                      </div>
                    )}
                    {messages.map((message) => {
                    const isOwnMessage = message.sender_id === user.id
                    const isEditingMessage = editingMessageID === message.id
                    const isPendingMessage = pendingMessageID === message.id
                    const deliveryStatus = messageDeliveryStatus(message)

                    return (
                      <Fragment key={message.id}>
                        {message.id === unreadDividerMessageID && (
                          <div className="unread-divider" ref={unreadDividerRef}>
                            <span>New messages</span>
                          </div>
                        )}
                        <article
                          className="message-bubble"
                          data-own={isOwnMessage}
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
                                {deliveryStatus && (
                                  <span
                                    className="delivery-status"
                                    data-status={deliveryStatus}
                                    aria-label={deliveryStatus === 'read' ? 'Read' : 'Sent'}
                                    title={deliveryStatus === 'read' ? 'Read' : 'Sent'}
                                  >
                                    <span className="delivery-ticks" aria-hidden="true">
                                      <span>✓</span>
                                      {deliveryStatus === 'read' && <span>✓</span>}
                                    </span>
                                    <span className="delivery-label">
                                      {deliveryStatus === 'read' ? 'Read' : 'Sent'}
                                    </span>
                                  </span>
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
                      </Fragment>
                    )
                    })}
                    <div ref={messageEndRef} />
                  </>
                )}
              </div>

              {messageError && <p className="form-message error">{messageError}</p>}
              {socketError && <p className="form-message warning">{socketError}</p>}
              {typingUserIDs.length > 0 && (
                <p className="typing-indicator">{typingLabel()}</p>
              )}
              {messages.length > 0 && !isAtMessageEnd && (
                <button
                  className="scroll-end-button"
                  type="button"
                  aria-label="Go to latest messages"
                  onClick={scrollToMessageEnd}
                >
                  ↓
                </button>
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

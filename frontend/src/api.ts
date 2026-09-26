const API_BASE_URL =
  import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080'

const WS_BASE_URL = API_BASE_URL.replace(/^http/, 'ws')

export type User = {
  id: string
  username: string
  email: string
  created_at: string
  updated_at: string
}

export type LoginResult = {
  user: User
  token: string
}

export type Conversation = {
  id: string
  type: 'direct' | 'room'
  name: string | null
  created_by: string | null
  created_at: string
  updated_at: string
  unread_count?: number
}

export type Message = {
  id: string
  conversation_id: string
  sender_id: string | null
  content: string
  created_at: string
  updated_at: string
}

export type DeletedMessage = {
  id: string
  conversation_id: string
}

export type RealtimeEvent =
  | {
      type: 'message.created'
      data: Message
    }
  | {
      type: 'message.updated'
      data: Message
    }
  | {
      type: 'message.deleted'
      data: DeletedMessage
    }
  | {
      type: 'typing.started' | 'typing.stopped'
      data: {
        conversation_id: string
        user_id: string
      }
    }

type RegisterInput = {
  username: string
  email: string
  password: string
}

type LoginInput = {
  email: string
  password: string
}

type CreateRoomInput = {
  name: string
}

type CreateDirectInput = {
  other_user_id: string
}

type CreateMessageInput = {
  content: string
}

type UpdateMessageInput = {
  content: string
}

type ApiErrorBody = {
  error?: string
}

export class ApiError extends Error {
  status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

export async function register(input: RegisterInput): Promise<User> {
  return request<User>('/auth/register', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export async function login(input: LoginInput): Promise<LoginResult> {
  return request<LoginResult>('/auth/login', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export async function me(token: string): Promise<User> {
  return request<User>('/auth/me', {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  })
}

export async function listConversations(token: string): Promise<Conversation[]> {
  return request<Conversation[]>('/conversations', {
    headers: authHeaders(token),
  })
}

export async function createRoom(
  token: string,
  input: CreateRoomInput,
): Promise<Conversation> {
  return request<Conversation>('/conversations/rooms', {
    method: 'POST',
    headers: authHeaders(token),
    body: JSON.stringify(input),
  })
}

export async function createDirect(
  token: string,
  input: CreateDirectInput,
): Promise<Conversation> {
  return request<Conversation>('/conversations/direct', {
    method: 'POST',
    headers: authHeaders(token),
    body: JSON.stringify(input),
  })
}

export async function listMessages(
  token: string,
  conversationID: string,
): Promise<Message[]> {
  return request<Message[]>(`/conversations/${conversationID}/messages`, {
    headers: authHeaders(token),
  })
}

export async function createMessage(
  token: string,
  conversationID: string,
  input: CreateMessageInput,
): Promise<Message> {
  return request<Message>(`/conversations/${conversationID}/messages`, {
    method: 'POST',
    headers: authHeaders(token),
    body: JSON.stringify(input),
  })
}

export async function updateMessage(
  token: string,
  conversationID: string,
  messageID: string,
  input: UpdateMessageInput,
): Promise<Message> {
  return request<Message>(
    `/conversations/${conversationID}/messages/${messageID}`,
    {
      method: 'PATCH',
      headers: authHeaders(token),
      body: JSON.stringify(input),
    },
  )
}

export async function deleteMessage(
  token: string,
  conversationID: string,
  messageID: string,
): Promise<void> {
  await request<void>(`/conversations/${conversationID}/messages/${messageID}`, {
    method: 'DELETE',
    headers: authHeaders(token),
  })
}

export function conversationWebSocketURL(token: string, conversationID: string) {
  const params = new URLSearchParams({ token })
  return `${WS_BASE_URL}/ws/conversations/${conversationID}?${params.toString()}`
}

function authHeaders(token: string) {
  return {
    Authorization: `Bearer ${token}`,
  }
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  headers.set('Content-Type', 'application/json')

  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    headers,
  })

  if (!response.ok) {
    let message = 'Something went wrong'
    try {
      const body = (await response.json()) as ApiErrorBody
      if (body.error) {
        message = body.error
      }
    } catch {
      message = response.statusText || message
    }

    throw new ApiError(message, response.status)
  }

  return response.json() as Promise<T>
}

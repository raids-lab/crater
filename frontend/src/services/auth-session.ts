import { ACCESS_TOKEN_KEY, REFRESH_TOKEN_KEY } from '@/utils/store'

// Only an opaque generation marker may leave the authentication layer.
export type AuthSession = object

interface AuthTokens {
  readonly accessToken: string | null
  readonly refreshToken: string | null
}

let current: (AuthTokens & { id: AuthSession }) | undefined

function readSession() {
  const accessToken = localStorage.getItem(ACCESS_TOKEN_KEY)
  const refreshToken = localStorage.getItem(REFRESH_TOKEN_KEY)
  if (!current || current.accessToken !== accessToken || current.refreshToken !== refreshToken) {
    current = { id: {}, accessToken, refreshToken }
  }
  return current
}

export function getAuthSession(): AuthSession {
  return readSession().id
}

export function getAuthTokens(): AuthTokens {
  return readSession()
}

export function requireAuthSession(session: AuthSession): AuthTokens {
  const active = readSession()
  if (active.id !== session) {
    throw new DOMException('Authentication session changed', 'AbortError')
  }
  return active
}

export function replaceAuthSession(accessToken: string, refreshToken: string) {
  localStorage.setItem(ACCESS_TOKEN_KEY, accessToken)
  localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken)
  // Login and account switches always start a new generation, even for identical tokens.
  current = { id: {}, accessToken, refreshToken }
}

export function clearAuthSession() {
  localStorage.removeItem(ACCESS_TOKEN_KEY)
  localStorage.removeItem(REFRESH_TOKEN_KEY)
  current = undefined
}

export function rotateAuthSession(session: AuthSession, accessToken: string, refreshToken: string) {
  requireAuthSession(session)
  localStorage.setItem(ACCESS_TOKEN_KEY, accessToken)
  localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken)
  // A refresh preserves the request owners for this session.
  current = { id: session, accessToken, refreshToken }
}

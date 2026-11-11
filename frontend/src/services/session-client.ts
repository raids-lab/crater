import ky, { HTTPError, type NormalizedOptions } from 'ky'

import {
  ERROR_ACCOUNT_DISABLED,
  ERROR_ACCOUNT_NOT_ACTIVATED,
  ERROR_TOKEN_EXPIRED,
  ERROR_TOKEN_INVALID,
  ERROR_USER_BANNED,
} from './error_code.ts'

const terminalCodes = new Set<number>([
  ERROR_TOKEN_EXPIRED,
  ERROR_TOKEN_INVALID,
  ERROR_ACCOUNT_NOT_ACTIVATED,
  ERROR_ACCOUNT_DISABLED,
  ERROR_USER_BANNED,
])

interface SessionOptions {
  prefixUrl: string
  getAccessToken: () => string | null
  getRefreshToken: () => string | null
  saveTokens: (access: string, refresh: string) => void
  onInvalid: (error: HTTPError) => void
  fetch?: typeof fetch
  timeout?: number
}

// One refresh serves concurrent expired requests. Each original request is replayed
// at most once; transport failures leave the stored session available for retry.
export function createSessionClient(settings: SessionOptions) {
  const basic = ky.create({
    prefixUrl: settings.prefixUrl,
    retry: 0,
    timeout: settings.timeout ?? 10000,
    ...(settings.fetch ? { fetch: settings.fetch } : {}),
  })
  // Other APIs reuse UserBanned for job-only restrictions. Those do not revoke
  // platform login; only the session check/refresh can establish a platform ban.
  const rejectsSession = (code: number | undefined, request: Request) =>
    terminalCodes.has(code ?? 0) &&
    (code !== ERROR_USER_BANNED || /\/auth\/check$/.test(new URL(request.url).pathname))
  interface Session {
    access: string | null
    refresh: string | null
    refreshing?: Promise<string>
  }
  let session: Session | undefined
  const currentSession = () => {
    const access = settings.getAccessToken()
    const refresh = settings.getRefreshToken()
    if (!session || session.access !== access || session.refresh !== refresh)
      session = { access, refresh }
    return session
  }
  const requestSessions = new WeakMap<NormalizedOptions, Session>()
  const notify = (error: HTTPError, data: unknown) => {
    Object.assign(error, { data })
    settings.onInvalid(error)
  }
  const refresh = (request: Request, options: NormalizedOptions, owner: Session) => {
    if (!owner.refreshing) {
      owner.refreshing = (async () => {
        const original = owner.refresh
        if (!original) {
          const error = new HTTPError(
            new Response(JSON.stringify({ code: ERROR_TOKEN_INVALID }), { status: 401 }),
            request,
            options
          )
          notify(error, { code: ERROR_TOKEN_INVALID })
          throw error
        }
        try {
          const response = await basic
            .post('auth/refresh', { json: { refreshToken: original } })
            .json<{ data: { accessToken: string; refreshToken: string } }>()
          if (!response.data?.accessToken || !response.data.refreshToken)
            throw new Error('Incomplete session response')
          // A logout or a newer login while refreshing must not restore the old session.
          if (currentSession() !== owner) throw new Error('Session changed during renewal')
          settings.saveTokens(response.data.accessToken, response.data.refreshToken)
          owner.access = response.data.accessToken
          owner.refresh = response.data.refreshToken
          return response.data.accessToken
        } catch (error) {
          if (error instanceof HTTPError && currentSession() === owner) {
            const data = await error.response
              .clone()
              .json()
              .catch(() => undefined)
            if (terminalCodes.has(data?.code) && currentSession() === owner) notify(error, data)
          }
          throw error
        }
      })().finally(() => {
        owner.refreshing = undefined
      })
    }
    return owner.refreshing
  }
  return basic.extend({
    hooks: {
      beforeRequest: [
        (request, options) => {
          const owner = currentSession()
          requestSessions.set(options, owner)
          const token = owner.access
          if (token) request.headers.set('Authorization', `Bearer ${token}`)
        },
      ],
      afterResponse: [
        async (request, options, response) => {
          if (
            response.ok ||
            /\/auth\/(login|signup|mode|refresh)$/.test(new URL(request.url).pathname)
          )
            return response
          const data = await response
            .clone()
            .json()
            .catch(() => undefined)
          const sentToken = request.headers.get('Authorization')
          const owner = requestSessions.get(options)
          // Only this client's renewal may keep a request in the same session.
          // Login, logout, account switches and external token changes end it.
          if (!owner || currentSession() !== owner) return response
          if (response.status === 401 && data?.code === ERROR_TOKEN_EXPIRED) {
            // A late response using the previous access token can reuse an already renewed one.
            const current = settings.getAccessToken()
            const token =
              current && sentToken !== `Bearer ${current}`
                ? current
                : await refresh(request, options, owner)
            if (currentSession() !== owner) return response
            request.headers.set('Authorization', `Bearer ${token}`)
            const retried = await basic(request, {
              ...options,
              prefixUrl: undefined,
              hooks: {},
              headers: request.headers,
              throwHttpErrors: false,
            })
            if (!retried.ok) {
              const failure = await retried
                .clone()
                .json()
                .catch(() => undefined)
              if (
                rejectsSession(failure?.code, request) &&
                currentSession() === owner &&
                owner.access === token
              )
                notify(new HTTPError(retried, request, options), failure)
            }
            return retried
          }
          if (
            rejectsSession(data?.code, request) &&
            sentToken === `Bearer ${settings.getAccessToken()}`
          )
            notify(new HTTPError(response, request, options), data)
          return response
        },
      ],
    },
  })
}

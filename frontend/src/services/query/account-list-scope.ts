import type { IUserContext } from '@/services/api/auth'

export function getAccountListScope(userId?: number, context?: IUserContext) {
  if (!userId || !context?.queue) return undefined

  // The server resolves the unique queue name to the account ID. The display nickname is separate.
  const { queue, roleQueue, rolePlatform, accessQueue, accessPublic } = context
  return JSON.stringify([userId, queue, roleQueue, rolePlatform, accessQueue, accessPublic])
}

export function accountListQueryKey(
  prefix: readonly string[],
  scope: string,
  ...params: readonly unknown[]
) {
  return [...prefix, scope, ...params] as const
}

export function assertAccountListScope(isCurrentScope: () => boolean) {
  if (!isCurrentScope()) throw new DOMException('Account changed', 'AbortError')
}

export function accountListQueryOptions<T>(
  queryKey: readonly unknown[],
  isCurrentScope: () => boolean,
  request: (signal: AbortSignal) => Promise<T>
) {
  return {
    // eslint-disable-next-line @tanstack/query/exhaustive-deps -- The key carries the scope; the callback only guards request lifetime.
    queryKey,
    queryFn: async ({ signal }: { signal: AbortSignal }) => {
      signal.throwIfAborted()
      assertAccountListScope(isCurrentScope)

      const data = await request(signal)
      signal.throwIfAborted()
      assertAccountListScope(isCurrentScope)
      return data
    },
  }
}

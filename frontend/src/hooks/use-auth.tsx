import { useQueryClient } from '@tanstack/react-query'
import { useAtom, useSetAtom } from 'jotai'
import React, { createContext, useContext, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import Loading from '@/components/placeholder/loading-spinner'

import { IAuthResponse, ILogin, IUserContext, apiCheckToken, apiLogin } from '@/services/api/auth'
import { getAuthErrorMessage } from '@/services/auth-errors'
import {
  clearAuthSession,
  getAuthSession,
  replaceAuthSession,
  requireAuthSession,
} from '@/services/auth-session'
import { IResponse } from '@/services/types'

import { logger } from '@/utils/loglevel'
import {
  UserInfo,
  atomBackendVersion,
  atomUserContext,
  atomUserInfo,
  globalLastView,
  useResetStore,
} from '@/utils/store'

export interface RouterAuthState {
  isAuthenticated: boolean
  user?: UserInfo | null
  context?: IUserContext | null
  login: (auth: ILogin) => Promise<IResponse<IAuthResponse>>
  logout: () => void
}

const AuthContext = createContext<RouterAuthState | undefined>(undefined)

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient()
  const [user, setUser] = useAtom(atomUserInfo)
  const [context, setContext] = useAtom(atomUserContext)
  const [, setBackendVersion] = useAtom(atomBackendVersion)
  const [isAuthenticated, setIsAuthenticated] = useState(false)
  const [isLoading, setIsLoading] = useState(true)
  const { resetAll } = useResetStore()
  const setLastView = useSetAtom(globalLastView)
  const { t } = useTranslation()

  // Restore auth state on app load
  useEffect(() => {
    const session = getAuthSession()
    apiCheckToken()
      .then(({ data }) => {
        if (getAuthSession() !== session) return
        if (data == undefined || data == null) {
          // No auth data means user is not authenticated
          logger.info('No auth data found, user not authenticated')
          setUser(undefined)
          setContext(undefined)
          setBackendVersion(undefined)
          setIsAuthenticated(false)
          return
        }

        // User is authenticated, set all data
        setUser({ ...data.user, space: data.context.space })
        setContext(data.context)
        setBackendVersion(data.version)
        setIsAuthenticated(true)
      })
      .catch((error) => {
        if (getAuthSession() !== session) return
        logger.error('Failed to check auth token:', error)
        toast.error(getAuthErrorMessage(error))
        setUser(undefined)
        setContext(undefined)
        setBackendVersion(undefined)
        setIsAuthenticated(false)
      })
      .finally(() => {
        setIsLoading(false)
      })
  }, [setContext, setUser, setBackendVersion])

  // Show loading state while checking auth
  if (isLoading) {
    return <Loading />
  }

  const login = async (auth: ILogin) => {
    const previous = getAuthSession()
    const res = await apiLogin(auth)
    requireAuthSession(previous)
    const { accessToken, refreshToken } = res.data
    replaceAuthSession(accessToken, refreshToken)
    const session = getAuthSession()

    try {
      // Fetch session metadata through the existing check API after saving the tokens.
      const { data } = await apiCheckToken()
      requireAuthSession(session)
      if (data == null) {
        throw new Error('No authenticated session returned by auth/check')
      }
      setUser({ ...data.user, space: data.context.space })
      setContext(data.context)
      setBackendVersion(data.version)
      setIsAuthenticated(true)
      return { ...res, data: { ...res.data, user: data.user, context: data.context } }
    } catch (error) {
      if (getAuthSession() === session) {
        clearAuthSession()
        setUser(undefined)
        setContext(undefined)
        setBackendVersion(undefined)
        setIsAuthenticated(false)
      }
      throw error
    }
  }

  const logout = () => {
    setIsAuthenticated(false)
    setLastView('portal')
    clearAuthSession()
    setUser(undefined)
    setContext(undefined)
    setBackendVersion(undefined)
    queryClient.clear()
    resetAll()
    toast.success(t('navUser.loggedOut'))
  }

  return (
    <AuthContext.Provider value={{ isAuthenticated, user, context, login, logout }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const context = useContext(AuthContext)
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}

import { useQueryClient } from '@tanstack/react-query'
import { useAtom, useSetAtom } from 'jotai'
import React, { createContext, useContext, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import Loading from '@/components/placeholder/loading-spinner'

import { IAuthResponse, ILogin, IUserContext, apiCheckToken, apiLogin } from '@/services/api/auth'
import { IResponse } from '@/services/types'

import { logger } from '@/utils/loglevel'
import {
  ACCESS_TOKEN_KEY,
  REFRESH_TOKEN_KEY,
  UserInfo,
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
  const [isAuthenticated, setIsAuthenticated] = useState(false)
  const [isLoading, setIsLoading] = useState(true)
  const { resetAll } = useResetStore()
  const setLastView = useSetAtom(globalLastView)
  const { t } = useTranslation()

  // Restore auth state on app load
  useEffect(() => {
    apiCheckToken()
      .then(({ data }) => {
        if (data == undefined) {
          throw new Error('No auth data found')
        }
        setUser({ ...data.user, space: data.context.space })
        setContext(data.context)
        setIsAuthenticated(true)
      })
      .catch((error) => {
        logger.error('Failed to check auth token:', error)
        setUser(undefined)
        setContext(undefined)
        setIsAuthenticated(false)
      })
      .finally(() => {
        setIsLoading(false)
      })
  }, [setContext, setUser])

  // Show loading state while checking auth
  if (isLoading) {
    return <Loading />
  }

  const login = async (auth: ILogin) => {
    return apiLogin(auth).then((res) => {
      const { accessToken, refreshToken, ...data } = res.data
      localStorage.setItem(ACCESS_TOKEN_KEY, accessToken)
      localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken)
      setUser({ ...data.user, space: data.context.space })
      setContext(data.context)
      setIsAuthenticated(true)
      return res
    })
  }

  const logout = () => {
    setIsAuthenticated(false)
    setLastView('portal')
    localStorage.removeItem(ACCESS_TOKEN_KEY)
    localStorage.removeItem(REFRESH_TOKEN_KEY)
    setUser(undefined)
    setContext(undefined)
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

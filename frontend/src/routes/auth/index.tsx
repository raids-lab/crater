/**
 * Copyright 2025 RAIDS Lab
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import { createFileRoute, redirect } from '@tanstack/react-router'
import { useState } from 'react'
import { toast } from 'sonner'

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'

import DocsButton from '@/components/button/docs-button'
import CraterIcon from '@/components/icon/crater-icon'
import CraterText from '@/components/icon/crater-text'
import NotFound from '@/components/placeholder/not-found'

import { AuthMode } from '@/services/api/auth'
import { queryAuthMode } from '@/services/query/auth'

import { useTheme } from '@/utils/theme'

import { ForgotPasswordForm } from './-components/forgot-password-form'
import { LoginForm } from './-components/login-form'
import { SignupForm } from './-components/signup-form'

export const Route = createFileRoute('/auth/')({
  validateSearch: (search) => ({
    redirect: (search.redirect as string) || undefined,
    token: (search.token as string) || undefined,
  }),
  beforeLoad: ({ context, search }) => {
    // Redirect if already authenticated
    if (context.auth.isAuthenticated && !!search.redirect) {
      throw redirect({ to: search.redirect })
    }
  },
  loader: async ({ context: { queryClient } }) => {
    return queryClient
      .ensureQueryData(queryAuthMode)
      .then((res) => {
        // ensureQueryData returns the raw result from queryFn (IResponse<IAuthModeResponse>)
        // We need to manually access .data here
        return {
          enableLdap: res.data?.enableLdap ?? false,
          enableNormalLogin: res.data?.enableNormalLogin ?? false,
          enableNormalRegister: res.data?.enableNormalRegister ?? false,
        }
      })
      .catch(() => {
        return {
          enableLdap: false,
          enableNormalLogin: true,
          enableNormalRegister: false,
        }
      })
  },
  component: LoginPage,
  notFoundComponent: () => <NotFound />,
})

function LoginPage() {
  const searchParams = Route.useSearch()
  const { auth } = Route.useRouteContext()
  const [showSignup, setShowSignup] = useState(false)
  const [showForgotPassword, setShowForgotPassword] = useState(false)
  const [showRegisterDialog, setShowRegisterDialog] = useState(false)
  const [registerDialogType, setRegisterDialogType] = useState<'ldap' | 'normal_disabled'>('ldap')
  const { theme, setTheme } = useTheme()
  const { enableLdap, enableNormalLogin, enableNormalRegister } = Route.useLoaderData()

  // Ensure selectedMode is one of enabled modes, preferring LDAP as default if enabled
  const [selectedMode, setSelectedMode] = useState<AuthMode>(() => {
    if (enableLdap) return AuthMode.LDAP
    return AuthMode.NORMAL
  })

  // Calculate if we should show mode switcher
  const showSwitcher = enableLdap && enableNormalLogin

  // Handle mode switching
  const handleModeChange = (newMode: string) => {
    const mode = newMode as AuthMode
    setSelectedMode(mode)
    setShowSignup(false)
    setShowForgotPassword(false)
  }

  // Handle registration button click
  const handleRegisterClick = () => {
    if (selectedMode === AuthMode.LDAP) {
      setRegisterDialogType('ldap')
      setShowRegisterDialog(true)
    } else {
      if (enableNormalRegister) {
        setShowSignup(true)
        setShowForgotPassword(false)
      } else {
        setRegisterDialogType('normal_disabled')
        setShowRegisterDialog(true)
      }
    }
  }

  // Handle forgot password button click
  const handleForgotPasswordClick = () => {
    if (selectedMode === AuthMode.LDAP) {
      toast.info('请联系平台管理员协助重置密码')
    } else {
      setShowForgotPassword(true)
      setShowSignup(false)
    }
  }

  // 返回登录表单
  const handleBackToLogin = () => {
    setShowSignup(false)
    setShowForgotPassword(false)
  }

  return (
    <div className="h-screen w-full lg:grid lg:grid-cols-2">
      {/* 左侧部分 */}
      <div className="bg-primary hidden lg:block dark:bg-slate-800/70">
        <div className="relative h-full w-full">
          {/* 顶部Logo */}
          <div
            className="absolute top-10 left-10 z-20 flex items-center text-lg font-medium"
            title="Switch signup and login"
          >
            <button
              className="flex h-14 w-full flex-row items-center justify-center text-white"
              onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
            >
              <CraterIcon className="mr-1.5 h-8 w-8" />
              <CraterText className="h-4" />
            </button>
          </div>
          {/* 底部版权信息 */}
          <div className="absolute bottom-10 left-10 z-20">
            <blockquote className="space-y-2">
              <footer className="text-sm text-white/80">Copyright @ RAIDS Lab</footer>
            </blockquote>
          </div>
          {/* 中间文字内容 */}
          <div className="relative flex h-full items-center justify-center">
            <div className="z-10 px-6 py-8 text-left text-white lg:px-16 lg:py-12">
              <h1 className="mb-4 text-5xl leading-tight font-semibold">
                <span className="dark:text-primary">欢迎体验</span>
                <br />
                异构云资源混合调度
                <br />
                与智能运维平台
              </h1>
              <DocsButton
                variant="ghost"
                className="dark:bg-primary dark:text-primary-foreground dark:hover:bg-primary/85 dark:hover:text-primary-foreground bg-white text-black hover:bg-slate-200 hover:text-black"
                title="平台文档"
                url=""
              />
            </div>
          </div>
        </div>
      </div>
      {/* 右侧表单部分 */}
      <div className="flex items-center justify-center py-12">
        {showSignup && selectedMode === AuthMode.NORMAL ? (
          <div className="mx-auto w-[350px] space-y-6">
            <div className="space-y-2 text-center">
              <h1 className="text-3xl font-bold">用户注册</h1>
              <p className="text-muted-foreground text-sm">注册您在 Crater 平台的账号</p>
            </div>
            <SignupForm />
            <div className="text-muted-foreground text-center text-sm">
              已有账号？
              <button onClick={handleBackToLogin} className="underline">
                立即登录
              </button>
            </div>
          </div>
        ) : showForgotPassword && selectedMode === AuthMode.NORMAL ? (
          <div className="mx-auto w-[350px] space-y-6">
            <div className="space-y-2 text-center">
              <h1 className="text-3xl font-bold">重置密码</h1>
              <p className="text-muted-foreground text-sm">我们将向您的邮箱发送密码重置链接</p>
            </div>
            <ForgotPasswordForm />
            <div className="text-muted-foreground text-center text-sm">
              想起密码了？
              <button onClick={handleBackToLogin} className="underline">
                返回登录
              </button>
            </div>
          </div>
        ) : (
          <div className="mx-auto w-[350px] space-y-6">
            <div className="space-y-2 text-center">
              <h1 className="text-3xl font-bold">用户登录</h1>
              <p className="text-muted-foreground text-sm">
                {selectedMode === AuthMode.LDAP
                  ? '已接入 LDAP 统一身份认证'
                  : '请输入您的账号和密码'}
              </p>
            </div>

            {showSwitcher && (
              <Tabs value={selectedMode} onValueChange={handleModeChange} className="w-full">
                <TabsList className="grid w-full grid-cols-2">
                  <TabsTrigger value={AuthMode.LDAP}>LDAP 登录</TabsTrigger>
                  <TabsTrigger value={AuthMode.NORMAL}>普通登录</TabsTrigger>
                </TabsList>
              </Tabs>
            )}

            <LoginForm
              searchParams={searchParams}
              login={auth.login}
              authMode={selectedMode}
              onForgotPasswordClick={handleForgotPasswordClick}
            />
            <div className="text-muted-foreground text-center text-sm">
              还没有账号？
              <button onClick={handleRegisterClick} className="underline">
                立即注册
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Registration guide dialog */}
      <AlertDialog open={showRegisterDialog} onOpenChange={setShowRegisterDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {registerDialogType === 'ldap' ? 'LDAP 账号登录说明' : '注册功能已禁用'}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {registerDialogType === 'ldap'
                ? '平台已接入 LDAP 统一身份认证。如果您拥有 LDAP 账号，可以直接在登录页面使用该账号及密码登录，系统将自动为您创建平台账户，无需进行额外的注册操作。'
                : '当前平台已禁用普通用户自主注册功能。请联系系统管理员协助为您创建账号，或者申请打开注册功能。'}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogAction onClick={() => setShowRegisterDialog(false)}>
              知道了
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

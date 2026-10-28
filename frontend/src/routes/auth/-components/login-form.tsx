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
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useAtomValue, useSetAtom } from 'jotai'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { toast } from 'sonner'
import { z } from 'zod'

import { Checkbox } from '@/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'

import LoadableButton from '@/components/button/loadable-button'

import { AuthMode, IAuthResponse, ILogin, Role } from '@/services/api/auth'
import { getAuthErrorMessage } from '@/services/auth-errors'
import { authMutationOptions } from '@/services/auth-mutation'
import { getApiErrorData, markApiErrorHandled } from '@/services/client'
import {
  ERROR_INVALID_CREDENTIALS,
  ERROR_MUST_REGISTER,
  ERROR_REGISTER_NOT_FOUND,
} from '@/services/error_code'
import {
  LEGACY_ERROR_INVALID_CREDENTIALS,
  LEGACY_ERROR_LDAP_USER_NOT_FOUND,
  LEGACY_ERROR_MUST_REGISTER,
} from '@/services/error_code_legacy'
import { IResponse } from '@/services/types'

import { atomPrivacyAccepted, atomUserContext, atomUserInfo, useResetStore } from '@/utils/store'

export type LoginSearch = {
  redirect?: string
  token?: string
}

const formSchema = z.object({
  // LDAP accounts and passwords follow the directory's rules, not local signup restrictions.
  username: z.string().min(1, { message: '用户名不能为空' }),
  password: z.string().min(1, { message: '密码不能为空' }),
  // 必须勾选隐私政策,否则无法通过校验
  acceptPrivacy: z.boolean().refine((val) => val === true, {
    message: '请先阅读并同意隐私政策方可登录',
  }),
})

interface LoginFormProps {
  authMode: AuthMode
  ldapAlias?: string
  login: (auth: ILogin) => Promise<IResponse<IAuthResponse>>
  onForgotPasswordClick: () => void
  searchParams: LoginSearch
}

export function LoginForm({
  authMode,
  ldapAlias,
  login,
  onForgotPasswordClick,
  searchParams,
}: LoginFormProps) {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const setUserState = useSetAtom(atomUserInfo)
  const setAccount = useSetAtom(atomUserContext)
  const { resetAll } = useResetStore()

  const setPrivacyAccepted = useSetAtom(atomPrivacyAccepted)
  const privacyAccepted = useAtomValue(atomPrivacyAccepted)

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      username: '',
      password: '',
      acceptPrivacy: privacyAccepted,
    },
  })

  const { mutate: loginUser, status } = useMutation({
    ...authMutationOptions,
    mutationFn: (values: { auth: string; username?: string; password?: string; token?: string }) =>
      login({
        auth: values.auth,
        username: values.username,
        password: values.password,
        token: values.token,
      }),
    onSuccess: ({ data }) => {
      queryClient.removeQueries({ predicate: (query) => query.queryKey[0] !== 'authmode' })
      setUserState({
        ...data.user,
        space: data.context.space,
      })
      setAccount(data.context)
      // 清除 GitHub Star 卡片关闭状态，每次登录都显示
      localStorage.removeItem('github-star-card-dismissed')
      toast.success(
        `你好，${data.context.rolePlatform === Role.Admin ? '系统管理员' : '用户'}${data.user.nickname}`
      )
      navigate({ to: searchParams.redirect || '/portal', replace: true })
    },
    onError: (error) => {
      markApiErrorHandled(error)
      const errorData = getApiErrorData(error)
      switch (errorData?.code) {
        case ERROR_INVALID_CREDENTIALS:
        case LEGACY_ERROR_INVALID_CREDENTIALS:
          form.setError('password', {
            type: 'server',
            message: getAuthErrorMessage(error),
          })
          return
        case ERROR_REGISTER_NOT_FOUND:
        case LEGACY_ERROR_LDAP_USER_NOT_FOUND:
          form.setError('username', {
            type: 'server',
            message: getAuthErrorMessage(error),
          })
          return
        case ERROR_MUST_REGISTER:
        case LEGACY_ERROR_MUST_REGISTER:
          form.setError('root.server', {
            type: 'server',
            message: getAuthErrorMessage(error),
          })
          return
        default:
          form.setError('root.server', { type: 'server', message: getAuthErrorMessage(error) })
      }
    },
  })

  const onSubmit = (values: z.infer<typeof formSchema>) => {
    if (status !== 'pending') {
      // zod 已经保证 acceptPrivacy === true，走到这里就是已同意
      resetAll()
      loginUser({
        username: values.username,
        password: values.password,
        auth: authMode === AuthMode.LDAP ? 'ldap' : 'normal',
      })
    }
  }

  useEffect(() => {
    if (searchParams.token) {
      loginUser({
        auth: 'normal', // 实际上后端只要看到 token 就会拦截并报错
        token: searchParams.token,
      })
    }
  }, [searchParams.token, loginUser])

  return (
    <>
      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
          <FormField
            control={form.control}
            name="username"
            render={({ field }) => (
              <FormItem>
                <FormLabel>账号</FormLabel>
                <FormControl>
                  <Input autoComplete="username" {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="password"
            render={({ field }) => (
              <FormItem>
                <div className="flex items-center justify-between">
                  <FormLabel>密码</FormLabel>
                  <button
                    className="text-muted-foreground p-0 text-sm underline"
                    type="button"
                    tabIndex={-1}
                    onClick={onForgotPasswordClick}
                  >
                    忘记密码？
                  </button>
                </div>
                <FormControl>
                  <Input type="password" autoComplete="current-password" {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          {/* 必须同意隐私政策 */}
          <FormField
            control={form.control}
            name="acceptPrivacy"
            render={({ field }) => (
              <FormItem>
                <div className="flex items-start gap-2">
                  <FormControl>
                    <Checkbox
                      id="acceptPrivacy"
                      checked={field.value}
                      onCheckedChange={(checked) => {
                        const value = checked === true
                        field.onChange(value)
                        setPrivacyAccepted(value)
                      }}
                    />
                  </FormControl>
                  <p className="text-muted-foreground text-xs leading-snug">
                    我已阅读并同意 <PrivacyPolicyDialog />
                  </p>
                </div>
                <FormMessage />
              </FormItem>
            )}
          />

          {form.formState.errors.root?.server?.message && (
            <p role="alert" className="text-destructive text-sm">
              {form.formState.errors.root.server.message}
            </p>
          )}

          <LoadableButton
            isLoadingText="登录中"
            type="submit"
            className="w-full"
            isLoading={status === 'pending'}
          >
            {authMode === AuthMode.LDAP
              ? ldapAlias
                ? `${ldapAlias} 登录`
                : '统一身份登录'
              : '登录'}
          </LoadableButton>
        </form>
      </Form>
    </>
  )
}

function PrivacyPolicyDialog() {
  return (
    <Dialog>
      <DialogTrigger asChild>
        <button
          type="button"
          className="text-primary underline underline-offset-4"
          onClick={(e) => e.stopPropagation()}
        >
          《隐私政策》
        </button>
      </DialogTrigger>

      <DialogContent className="h-[70vh] max-w-[520px] sm:max-w-[640px]">
        <DialogHeader>
          <DialogTitle className="text-base font-semibold sm:text-lg">隐私政策</DialogTitle>
        </DialogHeader>

        {/* 固定高度 + 内框 + 更小字体 */}
        <ScrollArea className="mt-3 max-h-[55vh] pr-2">
          <div className="bg-muted/40 text-muted-foreground space-y-3 rounded-md px-4 py-3 text-[11px] leading-relaxed sm:text-xs">
            <section>
              <p>
                本平台用于提供集群算力管理、作业调度、镜像与数据管理、监控审计和相关技术服务。
                为向您提供上述服务并保障平台安全运行，我们将根据本隐私政策收集和使用与您相关的必要信息。
              </p>
            </section>

            <section>
              <h3 className="text-foreground font-semibold">一、适用范围</h3>
              <p>
                本隐私政策适用于您登录、访问和使用本平台 Web 控制台及其提供的相关服务，
                包括但不限于作业创建与管理、资源申请、数据与镜像管理、监控与审计页面等。
              </p>
            </section>

            <section>
              <h3 className="text-foreground font-semibold">二、我们收集和处理的信息</h3>
              <p>我们基于合法合规、最小必要的原则,收集和处理以下类别的信息：</p>
              <ul className="list-disc space-y-1 pl-5">
                <li>账户标识信息：如姓名、学号/工号、用户名、邮箱、所属机构或部门等；</li>
                <li>平台使用信息：如登录时间、登录 IP、登录方式、访问页面与操作记录等；</li>
                <li>
                  作业与资源信息：如作业名称、配置、参数、镜像、挂载路径、资源使用情况与日志摘要等；
                </li>
                <li>必要的设备与环境信息：如浏览器类型、语言设置等，用于兼容性与安全分析。</li>
              </ul>
            </section>

            <section>
              <h3 className="text-foreground font-semibold">三、信息使用目的</h3>
              <ul className="list-disc space-y-1 pl-5">
                <li>提供身份认证、访问控制与账号管理；</li>
                <li>提供作业调度、资源分配、镜像和数据管理等功能；</li>
                <li>统计与分析资源使用情况，用于容量规划、性能优化和计费/配额管理（如适用）；</li>
                <li>进行安全审计、风控监测、异常排查和故障定位；</li>
                <li>履行法律法规、监管要求及上级管理机构规定的义务。</li>
              </ul>
            </section>

            <section>
              <h3 className="text-foreground font-semibold">四、信息的保存与保护</h3>
              <ul className="list-disc space-y-1 pl-5">
                <li>在实现上述目的所必需的最短期间内保存相关信息；</li>
                <li>限制只有经过授权且履职必要的人员或系统组件可访问信息；</li>
                <li>采取合理的技术和管理措施，防止信息被未经授权访问、使用或泄露。</li>
              </ul>
            </section>

            <section>
              <h3 className="text-foreground font-semibold">五、信息的共享与提供</h3>
              <p>我们不会向无关第三方提供您的个人信息，除非：</p>
              <ul className="list-disc space-y-1 pl-5">
                <li>事先获得您的明确同意或授权；</li>
                <li>依据法律法规或有权机关的强制要求；</li>
                <li>
                  在所属单位或管理机构内部基于管理、审计、安全目的进行必要共享，且受保密约束。
                </li>
              </ul>
            </section>

            <section>
              <h3 className="text-foreground font-semibold">六、Cookie 与本地存储</h3>
              <p>
                为维持登录状态、保存必要配置和提升体验，本平台可能使用 Cookie 或本地存储。
                相关信息仅用于本平台服务，不用于与本服务无关的跨站跟踪。
              </p>
            </section>

            <section>
              <h3 className="text-foreground font-semibold">七、您的权利</h3>
              <p>在符合适用政策和管理规定的前提下，您有权：</p>
              <ul className="list-disc space-y-1 pl-5">
                <li>查询与查看与您账户相关的基本信息和作业记录；</li>
                <li>在信息存在错误时申请更正；</li>
                <li>就隐私政策执行向平台管理员进行咨询或投诉。</li>
              </ul>
            </section>

            <section>
              <h3 className="text-foreground font-semibold">八、本政策的更新</h3>
              <p>
                如本隐私政策有重大变更，我们将通过平台公告或其他适当方式予以提示。
                更新后的政策自公布之日起生效。
              </p>
            </section>

            <section>
              <h3 className="text-foreground font-semibold">九、同意条款</h3>
              <p>
                您在登录并使用本平台前勾选“我已阅读并同意《隐私政策》”，即视为您已充分阅读、理解并同意本隐私政策的全部内容。
                如您不同意本隐私政策，将无法登录或使用本平台服务。
              </p>
            </section>
          </div>
        </ScrollArea>
      </DialogContent>
    </Dialog>
  )
}

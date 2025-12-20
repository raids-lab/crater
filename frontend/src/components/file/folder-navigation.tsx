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
// i18n-processed-v1.1.0
// Modified code
import { useLocation, useNavigate } from '@tanstack/react-router'
import { useAtomValue } from 'jotai'
import { ArrowRight, Folder, HardDrive, UserRound, UsersRound } from 'lucide-react'
import { motion } from 'motion/react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { getFolderTitle } from '@/components/file/lazy-file-tree'
import PageTitle from '@/components/layout/page-title'

import { AccessMode, IUserContext } from '@/services/api/auth'
import { FileItem } from '@/services/api/file'

import { atomUserContext } from '@/utils/store'

import { cn } from '@/lib/utils'

import DocsButton from '../button/docs-button'

const isPublicFolder = (folder: string) => folder === 'public'

const isAccountFolder = (folder: string) => folder === 'account'

const isUserFolder = (folder: string) => folder === 'user'

const getFolderDescription = (folder: string, t: (key: string) => string) => {
  if (isPublicFolder(folder)) {
    return t('folderNavigation.folderDescriptions.public')
  } else if (isAccountFolder(folder)) {
    return t('folderNavigation.folderDescriptions.account')
  }
  return t('folderNavigation.folderDescriptions.user')
}

const getAccessMode = (folder: string, token?: IUserContext) => {
  if (!token) {
    return AccessMode.NotAllowed
  }
  if (isPublicFolder(folder)) {
    return token.accessPublic
  } else if (isAccountFolder(folder)) {
    return token.accessQueue
  }
  return AccessMode.ReadWrite
}

export default function FolderNavigation({
  data: rowData,
  isadmin,
}: {
  data?: FileItem[]
  isadmin: boolean
}) {
  const { t } = useTranslation()
  const { pathname } = useLocation()
  const navigate = useNavigate()
  const context = useAtomValue(atomUserContext)

  // 对文件夹进行排序，公共 -> 账户 -> 用户
  const sortFolders = (folders: FileItem[]) => {
    return folders.sort((a, b) => {
      if (isPublicFolder(a.name)) {
        return -1
      } else if (isAccountFolder(a.name) && isUserFolder(b.name)) {
        return -1
      }
      return 1
    })
  }

  const data = useMemo(() => sortFolders(rowData || []), [rowData])

  const themes = {
    public: {
      bg: 'bg-blue-50/50 dark:bg-blue-950/20',
      border: 'border-blue-100 dark:border-blue-900',
      iconBg: 'bg-blue-100 text-blue-600 dark:bg-blue-900 dark:text-blue-300',
      badge: 'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300',
      progressBg: 'bg-blue-100',
      progressBar: 'bg-blue-500',
      button:
        'hover:bg-blue-600 hover:border-blue-600 hover:text-white dark:hover:bg-blue-700 dark:hover:border-blue-700',
      accentText: 'text-blue-600 dark:text-blue-400',
    },
    account: {
      bg: 'bg-emerald-50/50 dark:bg-emerald-950/20',
      border: 'border-emerald-100 dark:border-emerald-900',
      iconBg: 'bg-emerald-100 text-emerald-600 dark:bg-emerald-900 dark:text-emerald-300',
      progressBg: 'bg-emerald-100',
      progressBar: 'bg-emerald-500',
      badge: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900 dark:text-emerald-300',
      button:
        'hover:bg-emerald-600 hover:border-emerald-600 hover:text-white dark:hover:bg-emerald-700 dark:hover:border-emerald-700',
      accentText: 'text-emerald-600 dark:text-emerald-400',
    },
    user: {
      bg: 'bg-purple-50/50 dark:bg-purple-950/20',
      border: 'border-purple-100 dark:border-purple-900',
      iconBg: 'bg-purple-100 text-purple-600 dark:bg-purple-900 dark:text-purple-300',
      badge: 'bg-purple-100 text-purple-700 dark:bg-purple-900 dark:text-purple-300',
      progressBg: 'bg-highlight-purple/20',
      progressBar: 'bg-highlight-purple',
      button:
        'hover:bg-purple-600 hover:border-purple-600 hover:text-white dark:hover:bg-purple-700 dark:hover:border-purple-700',
      accentText: 'text-purple-600 dark:text-purple-400',
    },
  }

  const getBadgeText = (folder: string, mode: AccessMode) => {
    if (isUserFolder(folder)) return t('folderNavigation.badge.private', '私有')
    if (mode === AccessMode.ReadOnly) return t('folderNavigation.badge.readOnly', '只读')
    if (mode === AccessMode.ReadWrite) return t('folderNavigation.badge.readWrite', '读写')
    return t('folderNavigation.badge.noAccess', '无权限')
  }

  const handleTitleNavigation = (name: string) => {
    if (isPublicFolder(name)) {
      if (isadmin) {
        navigate({ to: pathname + '/admin-public' })
      } else {
        navigate({ to: pathname + '/public' })
      }
    } else if (isAccountFolder(name)) {
      if (isadmin) {
        navigate({ to: `${pathname}/admin-account` })
      } else {
        navigate({ to: `${pathname}/account` })
      }
    } else {
      if (isadmin) {
        navigate({ to: `${pathname}/admin-user` })
      } else {
        navigate({ to: `${pathname}/user` })
      }
    }
  }

  return (
    <div>
      <PageTitle
        title={t('folderNavigation.pageTitle.title')}
        description={t('folderNavigation.pageTitle.description')}
      >
        {/* <DocsButton title="阅读文档" url="file/file/" /> */}
      </PageTitle>
      <div
        className={cn('mt-6 grid gap-6', {
          'grid-cols-1 md:grid-cols-2': data.length === 2,
          'grid-cols-1 md:grid-cols-3': data.length === 3,
        })}
      >
        {data.map((r, index) => {
          const type = isPublicFolder(r.name)
            ? 'public'
            : isAccountFolder(r.name)
              ? 'account'
              : 'user'
          const theme = themes[type]
          const Icon = type === 'public' ? HardDrive : type === 'account' ? UsersRound : UserRound
          const badgeText = getBadgeText(r.name, getAccessMode(r.name, context))

          return (
            <motion.div
              key={r.name}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.3, delay: index * 0.1 }}
              whileHover={{ y: -5 }}
            >
              <div
                className={cn(
                  'group bg-card relative flex flex-col overflow-hidden rounded-3xl border p-6 shadow-sm transition-all duration-300 hover:-translate-y-1 hover:shadow-xl'
                )}
              >
                {/* Large background number watermark */}
                <div className="pointer-events-none absolute -top-8 -right-6 font-sans text-[120px] font-bold opacity-[0.03] select-none">
                  {index + 1}
                </div>

                {/* Header Section */}
                <div className="relative z-10 mb-6 flex items-start justify-between">
                  <div className={cn('rounded-xl p-2 shadow-sm', theme.iconBg)}>
                    <Icon className="size-6" />
                  </div>
                  <span
                    className={cn(
                      'rounded-full px-3 py-1 text-xs font-semibold shadow-sm',
                      theme.badge
                    )}
                  >
                    {badgeText}
                  </span>
                </div>

                {/* Title & Desc */}
                <div className="relative z-10 mb-8 flex-1">
                  <h3 className="mb-3 text-xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
                    {getFolderTitle(t, r.name)}
                  </h3>
                  <p className="text-highlight-slate line-clamp-2 h-10 text-sm leading-relaxed">
                    {getFolderDescription(r.name, t)}
                  </p>
                </div>
                {/* Usage Metrics */}
                <div className={`rounded-2xl p-4 ${theme.bg} relative z-10 mb-6 border`}>
                  <div className="mb-2 flex items-baseline justify-between">
                    <div className="flex items-baseline gap-1">
                      <span className="text-foreground text-lg font-bold">???</span>
                      <span className="text-highlight-slate text-sm font-medium">GB</span>
                    </div>
                    <span className="text-highlight-slate text-xs">总 ??? GB</span>
                  </div>

                  <div
                    className={`h-2 w-full ${theme.progressBg} mb-2 overflow-hidden rounded-full`}
                  >
                    <div
                      className={`h-full ${theme.progressBar} rounded-full transition-all duration-700 ease-out`}
                      style={{ width: `50%` }}
                    />
                  </div>

                  <div className="flex items-center justify-between text-xs">
                    <span className="text-highlight-slate">??% 已使用</span>
                    <span className="text-highlight-slate">{r.size} 个文件</span>
                  </div>
                </div>
                {/* Action Button */}
                <button
                  className={cn(
                    'group/btn relative flex h-10 w-full items-center justify-center gap-2 rounded-md border border-slate-200 bg-white px-4 py-3 text-sm font-medium text-slate-600 shadow-sm transition-all duration-200 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-300',
                    theme.button
                  )}
                  onClick={() => handleTitleNavigation(r.name)}
                >
                  <span>{t('folderNavigation.viewButton', { folder: '' })}</span>
                  <ArrowRight className="h-4 w-4 transition-transform group-hover/btn:translate-x-1" />
                </button>
              </div>
            </motion.div>
          )
        })}
      </div>

      {rowData != undefined && data.length === 0 && (
        <div className="py-12 text-center">
          <Folder className="text-muted-foreground/50 mx-auto mb-4 size-12" />
          <h3 className="mb-2 text-xl font-medium">{t('folderNavigation.noFolders.title')}</h3>
          <p className="text-muted-foreground">{t('folderNavigation.noFolders.description')}</p>
        </div>
      )}
    </div>
  )
}

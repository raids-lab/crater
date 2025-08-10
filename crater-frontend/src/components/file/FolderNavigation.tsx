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
import { ChevronRight, Folder, FolderOpen, LogInIcon } from 'lucide-react'
import { motion } from 'motion/react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'

import { getFolderTitle } from '@/components/file/LazyFileTree'
import PageTitle from '@/components/layout/page-title'

import { AccessMode, IUserContext } from '@/services/api/auth'
import { FileItem } from '@/services/api/file'

import { atomUserContext } from '@/utils/store'

import { cn } from '@/lib/utils'

import UserAccessBadge from '../badge/UserAccessBadge'

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
  const [hoveredFolder, setHoveredFolder] = useState<string | null>(null)
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

  // 为每个文件夹分配一个固定的颜色
  const getFolderColor = (index: number) => {
    const colors = [
      {
        bg: 'bg-blue-50 dark:bg-blue-950/30',
        border: 'border-blue-200 dark:border-blue-800',
        icon: 'text-blue-600 dark:text-blue-400',
      },
      {
        bg: 'bg-green-50 dark:bg-green-950/30',
        border: 'border-green-200 dark:border-green-800',
        icon: 'text-green-600 dark:text-green-400',
      },
      {
        bg: 'bg-purple-50 dark:bg-purple-950/30',
        border: 'border-purple-200 dark:border-purple-800',
        icon: 'text-purple-600 dark:text-purple-400',
      },
    ]

    return colors[index % colors.length]
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
      />
      <div
        className={cn('mt-6 grid gap-6', {
          'grid-cols-1 md:grid-cols-2': data.length === 2,
          'grid-cols-1 md:grid-cols-3': data.length === 3,
        })}
      >
        {data.map((r, index) => {
          const colors = getFolderColor(index)

          return (
            <motion.div
              key={r.name}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.3, delay: index * 0.1 }}
              whileHover={{ y: -5 }}
            >
              <Card
                className={cn(
                  'group h-full transition-all duration-300',
                  'dark:hover:shadow-primary/10 border-2 hover:shadow-lg',
                  colors.bg,
                  colors.border,
                  hoveredFolder === r.name && 'ring-primary/20 ring-2'
                )}
                onMouseEnter={() => setHoveredFolder(r.name)}
                onMouseLeave={() => setHoveredFolder(null)}
              >
                <CardHeader className="pb-4">
                  <div className="mb-2 flex items-center gap-3">
                    {hoveredFolder === r.name ? (
                      <FolderOpen className={cn('size-6', colors.icon)} />
                    ) : (
                      <Folder className={cn('size-6', colors.icon)} />
                    )}
                    <CardTitle className="flex flex-row items-center gap-2 text-xl">
                      {getFolderTitle(t, r.name)}
                      <UserAccessBadge access={getAccessMode(r.name, context).toString()} />
                    </CardTitle>
                  </div>
                  <CardDescription className="leading-relaxed text-balance">
                    {getFolderDescription(r.name, t)}
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <div
                    className={cn(
                      'flex h-24 items-center justify-center rounded-md',
                      'bg-white/50 backdrop-blur-xs dark:bg-white/5'
                    )}
                  >
                    <div
                      className={cn(
                        'text-4xl transition-opacity',
                        'opacity-20 group-hover:opacity-30',
                        colors.icon
                      )}
                    >
                      {index + 1}
                    </div>
                  </div>
                </CardContent>
                <CardFooter className="pt-4">
                  <Button
                    className="group w-full"
                    onClick={() => handleTitleNavigation(r.name)}
                    variant="outline"
                  >
                    <LogInIcon className="mr-2 size-4 transition-transform group-hover:translate-x-1" />
                    {t('folderNavigation.viewButton', {
                      folder: getFolderTitle(t, r.name),
                    })}
                    <ChevronRight className="ml-auto size-4 transition-transform group-hover:translate-x-1" />
                  </Button>
                </CardFooter>
              </Card>
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

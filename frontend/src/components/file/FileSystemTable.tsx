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
import { useTranslation } from 'react-i18next'
import { useNavigate, useLocation } from 'react-router-dom'
import { useState, useEffect, useMemo, useRef, useCallback } from 'react'
import { DataTable } from '@/components/custom/DataTable'
import { useQuery, useQueryClient, useMutation } from '@tanstack/react-query'
import { DataTableColumnHeader } from '@/components/custom/DataTable/DataTableColumnHeader'
import { ColumnDef, Row } from '@tanstack/react-table'
import { DataTableToolbarConfig } from '@/components/custom/DataTable/DataTableToolbar'
import { Button } from '@/components/ui/button'
import { toast } from 'sonner'
import { TimeDistance } from '@/components/custom/TimeDistance'
import { FileSizeComponent } from '@/components/file/FileSize'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  DialogClose,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  ArrowLeftIcon,
  DownloadIcon,
  File,
  Folder,
  FolderPlusIcon,
  MoveIcon,
  Trash2,
} from 'lucide-react'
import { useAtomValue, useSetAtom } from 'jotai'
import { globalAccount, globalBreadCrumb } from '@/utils/store'
import { FileItem, MoveFile, apiFileDelete, apiMkdir, apiMoveFile } from '@/services/api/file'
import { ACCESS_TOKEN_KEY } from '@/utils/store'
import { showErrorToast } from '@/utils/toast'
import { getAdminFolderTitle, getFolderTitle } from '@/components/file/LazyFileTree'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui-custom/alert-dialog'
import FileUpload from '@/components/file/FileUpload'
import FolderNavigation from './FolderNavigation'
import { AxiosResponse } from 'axios'
import { IResponse } from '@/services/types'
import { FileSelectDialog } from './FileSelectDialog'
import TooltipButton from '../custom/TooltipButton'
import { configUrlApiBaseAtom } from '@/utils/store/config'
import { AccessMode } from '@/services/api/auth'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@radix-ui/react-tooltip'
import { BaseUserInfo, apiGetBaseUserInfo } from '@/services/api/user'
import { IAccount, apiAdminAccountList } from '@/services/api/account'
import { cn } from '@/lib/utils'

const getHeader = (key: string, t: (key: string) => string): string => {
  switch (key) {
    case 'name':
      return t('columns.name.header')
    case 'modifytime':
      return t('columns.modifytime.header')
    case 'size':
      return t('columns.size.header')
    default:
      return key
  }
}

const getToolbarConfig = (t: (key: string) => string): DataTableToolbarConfig => {
  return {
    filterInput: {
      placeholder: 'search.name.placeholder',
      key: 'name',
    },
    filterOptions: [],
    getHeader: (key) => getHeader(key, t),
  }
}

interface FilesystemTableProps {
  apiGetFiles: (path: string) => Promise<AxiosResponse<IResponse<FileItem[] | undefined>>>
  isadmin: boolean
}

const FileActions = ({
  apiBaseURL,
  deleteFile,
  moveFile,
  isDir,
  name,
  path,
  canShow,
}: {
  apiBaseURL: string
  deleteFile: (path: string) => void
  moveFile: ({ fileData, path }: { fileData: MoveFile; path: string }) => void
  isDir: boolean
  name: string
  path: string
  canShow: boolean
}) => {
  const [isMoveDialogOpen, setIsMoveDialogOpen] = useState(false)
  const { t } = useTranslation()
  return (
    <div className="flex flex-row space-x-1">
      {/* 下载按钮（仅文件显示）*/}
      {!isDir ? (
        <TooltipButton
          variant="outline"
          className="size-8 p-0 hover:text-sky-700"
          tooltipContent={t('fileActions.download.tooltip')}
          onClick={() => {
            const link = `${apiBaseURL}ss/download${path}/${name}`
            const o = new XMLHttpRequest()
            o.open('GET', link)
            o.responseType = 'blob'
            const token = localStorage.getItem(ACCESS_TOKEN_KEY)
            o.setRequestHeader('Authorization', 'Bearer ' + token)
            o.onload = function () {
              if (o.status == 200) {
                const content = o.response as string
                const a = document.createElement('a')
                a.style.display = 'none'
                a.download = name || ''
                const blob = new Blob([content])
                a.href = URL.createObjectURL(blob)
                document.body.appendChild(a)
                a.click()
                document.body.removeChild(a)
                toast.success(t('fileActions.download.success'))
              } else {
                toast.error(
                  t('fileActions.download.error', {
                    status: o.statusText,
                  })
                )
              }
            }
            o.send()
            toast.info(t('fileActions.download.inProgress'))
          }}
        >
          <DownloadIcon className="size-4" />
        </TooltipButton>
      ) : (
        <div className="size-8" />
      )}
      {/* 移动操作 （有读写权限显示）*/}
      {canShow ? (
        <AlertDialog open={isMoveDialogOpen} onOpenChange={setIsMoveDialogOpen}>
          <AlertDialogTrigger asChild>
            <div>
              <TooltipButton
                className="hover:text-destructive h-8 w-8 p-0"
                variant="outline"
                size="icon"
                tooltipContent={t('fileActions.move.tooltip', {
                  type: isDir ? 'folder' : 'file',
                })}
              >
                <MoveIcon size={16} strokeWidth={2} />
              </TooltipButton>
            </div>
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle className="text-lg">
                {t('fileActions.move.title', {
                  type: isDir ? 'folder' : 'file',
                })}
              </AlertDialogTitle>
              <AlertDialogDescription>
                <div className="text-foreground font-medium">
                  {t('fileActions.move.currentItem', {
                    name,
                  })}
                </div>
                <p className="text-sm">{t('fileActions.move.description')}</p>
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>{t('common.cancel')}</AlertDialogCancel>
              <AlertDialogAction asChild>
                <FileSelectDialog
                  value=""
                  handleSubmit={(item) => {
                    setIsMoveDialogOpen(false)
                    moveFile({
                      fileData: {
                        fileName: name,
                        dst: item.id + '/' + name,
                      },
                      path: `${path}/${name}`,
                    })
                  }}
                  isrw={true}
                  title="选择要移动到的位置"
                  disabled={false}
                  allowSelectFile={!isDir}
                />
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      ) : (
        <div className="size-8" />
      )}
      {/* 删除操作 （有读写权限显示）*/}
      {canShow ? (
        <AlertDialog>
          <AlertDialogTrigger asChild>
            <div>
              <TooltipButton
                className="hover:text-destructive h-8 w-8 p-0"
                variant="outline"
                size="icon"
                tooltipContent={t('fileActions.delete.tooltip', {
                  type: isDir ? 'folder' : 'file',
                })}
              >
                <Trash2 size={16} strokeWidth={2} />
              </TooltipButton>
            </div>
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>
                {t('fileActions.delete.title', {
                  type: isDir ? 'folder' : 'file',
                })}
              </AlertDialogTitle>
              <AlertDialogDescription>
                {t('fileActions.delete.description', {
                  type: isDir ? 'folder' : 'file',
                  name,
                })}
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>{t('common.cancel')}</AlertDialogCancel>
              <AlertDialogAction
                variant="destructive"
                onClick={() => {
                  deleteFile(path + '/' + name)
                }}
              >
                {t('common.delete')}
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      ) : (
        <div className="size-8" />
      )}
    </div>
  )
}

export function FileSystemTable({ apiGetFiles, isadmin }: FilesystemTableProps) {
  const { t } = useTranslation()
  const { pathname } = useLocation()
  const navigate = useNavigate()
  const [dirName, setDirName] = useState<string>('')
  const setBreadcrumb = useSetAtom(globalBreadCrumb)
  const apiBaseURL = useAtomValue(configUrlApiBaseAtom)

  const path = useMemo(() => {
    const basePattern = isadmin ? /^\/admin\/data\/filesystem/ : /^\/portal\/data\/filesystem/
    return pathname.replace(basePattern, '')
  }, [pathname, isadmin])

  useEffect(() => {
    const pathParts = pathname.split('/').filter(Boolean)
    const breadcrumb = pathParts.map((value, index) => {
      value = decodeURIComponent(value)
      if (!isadmin) {
        if (index == 0 && value == 'portal') {
          return {
            title: 'Portal',
          }
        } else if (index == 1 && value == 'data') {
          return {
            title: '数据管理',
            path: '/portal/data',
            isEmpty: true,
          }
        } else if (index == 2 && value == 'filesystem') {
          return {
            title: '文件系统',
            path: '/portal/data/filesystem',
          }
        } else if (index == 3) {
          return {
            title: getFolderTitle(t, value),
            path: `/portal/data/filesystem/${value}`,
          }
        }
      } else {
        if (index == 0 && value == 'admin') {
          return {
            title: 'Admin',
          }
        } else if (index == 1 && value == 'data') {
          return {
            title: '数据管理',
            path: '/admin/data',
            isEmpty: true,
          }
        } else if (index == 2 && value == 'filesystem') {
          return {
            title: '文件系统',
            path: '/admin/data/filesystem',
          }
        } else if (index == 3) {
          return {
            title: getAdminFolderTitle(t, value),
            path: `/admin/data/filesystem/${value}`,
          }
        }
      }

      return {
        title: value,
        path: `/${pathParts.slice(0, index + 1).join('/')}`,
      }
    })
    // 删除第一个元素
    breadcrumb.shift()
    // 将最后一个元素的 path 设置为 undefined
    if (breadcrumb.length > 1) {
      breadcrumb[breadcrumb.length - 1].path = undefined
    }
    setBreadcrumb(breadcrumb)
  }, [isadmin, pathname, setBreadcrumb, t])

  const backpath = useMemo(() => {
    return pathname.replace(/\/[^/]+$/, '')
  }, [pathname])

  const query = useQuery({
    queryKey: ['data', 'filesystem', path],
    queryFn: () => apiGetFiles(`${path}`),
    select: (res) => {
      return (
        res.data.data
          ?.map((r) => {
            return {
              name: r.name,
              modifytime: r.modifytime,
              isdir: r.isdir,
              size: r.size,
              sys: r.sys,
            }
          })
          .sort((a, b) => {
            if (a.isdir && !b.isdir) {
              return -1 // a在b之前
            } else if (!a.isdir && b.isdir) {
              return 1 // a在b之后
            } else {
              return a.name.localeCompare(b.name)
            }
          }) ?? []
      )
    },
    staleTime: 10 * 1000,
  })

  const userInfoQuery = useQuery({
    queryKey: ['users', 'baseinfo'],
    queryFn: () => {
      return apiGetBaseUserInfo()
    },
    select: (res) => res.data.data,
    staleTime: 5 * 60 * 1000,
    enabled: isadmin,
  })

  const userMap = useMemo(() => {
    // 添加类型断言确保数据安全
    if (!userInfoQuery.data || !Array.isArray(userInfoQuery.data)) {
      return new Map<string, BaseUserInfo>()
    }
    return new Map(userInfoQuery.data.map((u) => [u.space, u]))
  }, [userInfoQuery.data])

  const accountInfoQuery = useQuery({
    queryKey: ['accounts', 'baseinfo'],
    queryFn: () => {
      return apiAdminAccountList()
    },
    select: (res) => {
      return res.data.data
    },
    staleTime: 5 * 60 * 1000,
    enabled: isadmin,
  })
  const accountMap = useMemo(() => {
    // 添加类型断言确保数据安全
    if (!accountInfoQuery.data || !Array.isArray(accountInfoQuery.data)) {
      return new Map<string, IAccount>()
    }
    return new Map(accountInfoQuery.data.map((a) => [a.space, a]))
  }, [accountInfoQuery.data])

  const isRoot = useMemo(() => {
    const basePath = isadmin ? '/admin/data/filesystem' : '/portal/data/filesystem'
    return pathname === basePath
  }, [pathname, isadmin])

  const isAdminUserSpace = useMemo(() => {
    const basePath = '/admin/data/filesystem/admin-user'
    return pathname === basePath
  }, [pathname])

  const isAdminAccountSpace = useMemo(() => {
    const basePath = '/admin/data/filesystem/admin-account'
    return pathname === basePath
  }, [pathname])

  const { mutate: deleteFile } = useMutation({
    mutationFn: (req: string) => apiFileDelete(req),
    onSuccess: async () => {
      toast.success(t('fileActions.delete.success'))
      await queryClient.invalidateQueries({
        queryKey: ['data', 'filesystem', path],
      })
    },
  })

  const token = useAtomValue(globalAccount)
  const canShow = useMemo(() => {
    if (path.startsWith('/public')) {
      return token.accessPublic === AccessMode.ReadWrite
    }
    if (path.startsWith('/account')) {
      return token.accessQueue === AccessMode.ReadWrite // 根据实际权限规则调整
    }
    if (path.startsWith('/user') || path.startsWith('/admin')) {
      return true
    }
    return false // 默认不显示
  }, [path, token.accessPublic, token.accessQueue])

  const { mutate: moveFile } = useMutation({
    mutationFn: ({ fileData, path }: { fileData: MoveFile; path: string }) =>
      apiMoveFile(fileData, path),
    onSuccess: async () => {
      toast.success(t('fileActions.move.success'))
      await queryClient.invalidateQueries({
        queryKey: ['data', 'filesystem', path],
      })
    },
  })
  interface HasNicknameAndName {
    nickname: string
    name: string
  }

  const renderNameCell = useCallback(
    <T extends HasNicknameAndName>(
      row: Row<FileItem>,
      infoMap: Map<string, T>,
      isSpecialPath: boolean
    ) => {
      const targetInfo = infoMap.get(row.getValue('name'))
      const displayName = targetInfo?.nickname ?? row.getValue('name')

      if (!isSpecialPath) {
        // 默认渲染逻辑
        return row.original.isdir ? (
          <div className="flex items-center">
            <Folder className="mr-2 size-5 text-yellow-600 dark:text-yellow-400" />
            <Button
              onClick={() => navigate(pathname + '/' + row.original.name)}
              variant="link"
              className="text-secondary-foreground h-8 px-0 text-left font-normal"
            >
              {row.getValue('name')}
            </Button>
          </div>
        ) : (
          <div className="flex items-center">
            <File className="text-muted-foreground mr-2 size-5" />
            <span className="text-secondary-foreground">{row.getValue('name')}</span>
          </div>
        )
      }

      return (
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              {row.original.isdir ? (
                <Button
                  onClick={() => navigate(pathname + '/' + row.original.name)}
                  variant="link"
                  className="text-secondary-foreground hover:text-primary h-8 px-0 text-left font-normal"
                >
                  <div className="flex items-center">
                    <Folder className="mr-2 size-5 text-yellow-600 dark:text-yellow-400" />
                    {displayName}
                  </div>
                </Button>
              ) : (
                <div className="flex items-center">
                  <File className="text-muted-foreground mr-2 size-5" />
                  <span className="text-secondary-foreground">{displayName}</span>
                </div>
              )}
            </TooltipTrigger>
            {targetInfo && (
              <TooltipContent
                side="top"
                align="start"
                className={cn(
                  'bg-primary text-primary-foreground animate-in fade-in-0 zoom-in-95 data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2 z-50 w-fit rounded-md px-3 py-1.5 text-xs text-balance'
                )}
              >
                <p>
                  查看{displayName}
                  <span className="mx-0.5 font-mono">(@{targetInfo.name})</span>
                  的文件信息
                </p>
              </TooltipContent>
            )}
          </Tooltip>
        </TooltipProvider>
      )
    },
    [navigate, pathname]
  )

  const baseColumns = useMemo<ColumnDef<FileItem>[]>(() => {
    return [
      {
        accessorKey: 'modifytime',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('modifytime', t)} />
        ),
        cell: ({ row }) => {
          return <TimeDistance date={row.getValue('modifytime')}></TimeDistance>
        },
      },
      {
        accessorKey: 'size',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('size', t)} />
        ),
        cell: ({ row }) => {
          if (row.original.isdir) {
            return
          } else {
            return <FileSizeComponent size={row.getValue('size')} />
          }
        },
      },
      {
        id: 'actions',
        enableHiding: false,
        cell: ({ row }) => {
          const { isdir, name } = row.original

          return (
            <FileActions
              apiBaseURL={apiBaseURL}
              deleteFile={deleteFile}
              moveFile={moveFile}
              isDir={isdir}
              name={name}
              path={path}
              canShow={canShow}
            />
          )
        },
      },
    ]
  }, [apiBaseURL, deleteFile, moveFile, path, canShow, t])

  const nameColumn = useMemo<ColumnDef<FileItem>[]>(() => {
    return [
      {
        accessorKey: 'name',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('name', t)} />
        ),
        cell: ({ row }) => {
          if (isAdminAccountSpace) {
            return renderNameCell(row, accountMap, true)
          } else if (isAdminUserSpace) {
            return renderNameCell(row, userMap, true)
          }
          return renderNameCell(row, new Map(), false)
        },
      },
    ]
  }, [t, isAdminAccountSpace, isAdminUserSpace, renderNameCell, accountMap, userMap])
  const columns = [...nameColumn, ...baseColumns]

  const refInput2 = useRef<HTMLInputElement>(null)

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    e.persist()
    setDirName(e.target.value)
  }
  const queryClient = useQueryClient()

  const clientCreateDir = async (path: string) => {
    //console.log("dirName:" + dirName);
    if (dirName != '') {
      await apiMkdir(`${path}/${dirName}`)
        .then(() => {
          toast.success(t('fileActions.createFolder.success'))
        })
        .catch((error) => {
          showErrorToast(error)
        })
    }
  }
  const { mutate: CreateDir } = useMutation({
    mutationFn: () => clientCreateDir(path),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ['data', 'filesystem', path],
      })
    },
  })

  const handleReturnNavigation = (backpath: string) => {
    navigate(backpath)
  }

  return (
    <>
      {isRoot ? (
        <FolderNavigation data={query.data} isadmin={isadmin}></FolderNavigation>
      ) : (
        <DataTable
          storageKey="filesystem"
          query={query}
          columns={columns}
          toolbarConfig={getToolbarConfig(t)}
        >
          <TooltipButton
            variant="outline"
            size="icon"
            onClick={() => {
              handleReturnNavigation(backpath)
            }}
            className="h-8 w-8"
            disabled={isRoot}
            tooltipContent={t('fileActions.return.tooltip')}
          >
            <ArrowLeftIcon className="size-4" />
          </TooltipButton>
          <FileUpload uploadPath={path} disabled={isRoot}></FileUpload>
          <Dialog>
            <DialogTrigger asChild>
              <TooltipButton
                variant="outline"
                className="h-8 w-8"
                size="icon"
                disabled={isRoot}
                tooltipContent={t('fileActions.createFolder.tooltip')}
              >
                <FolderPlusIcon className="size-4" />
              </TooltipButton>
            </DialogTrigger>
            <DialogContent className="sm:max-w-[425px]">
              <DialogHeader>
                <DialogTitle>{t('fileActions.createFolder.title')}</DialogTitle>
                <DialogDescription>{t('fileActions.createFolder.description')}</DialogDescription>
              </DialogHeader>
              <div className="grid gap-4 py-4">
                <div className="grid grid-cols-4 items-center gap-4">
                  <Label htmlFor="name" className="text-right">
                    {t('fileActions.createFolder.nameLabel')}
                  </Label>
                  <Input
                    id="name"
                    type="text"
                    defaultValue=""
                    className="col-span-3"
                    ref={refInput2}
                    onChange={handleChange}
                  />
                </div>
              </div>
              <DialogClose>
                <DialogFooter>
                  <Button type="submit" onClick={() => CreateDir()}>
                    {t('common.create')}
                  </Button>
                </DialogFooter>
              </DialogClose>
            </DialogContent>
          </Dialog>
        </DataTable>
      )}
    </>
  )
}

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
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@radix-ui/react-tooltip'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useLocation, useNavigate, useRouter } from '@tanstack/react-router'
import { ColumnDef, Row } from '@tanstack/react-table'
import { useAtomValue, useSetAtom } from 'jotai'
import {
  ArrowLeftIcon,
  DownloadIcon,
  File,
  FileOutputIcon,
  Folder,
  FolderOutputIcon,
  FolderPlusIcon,
  Trash2,
} from 'lucide-react'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

import { TimeDistance } from '@/components/custom/TimeDistance'
import { FileSizeComponent } from '@/components/file/FileSize'
import FileUpload from '@/components/file/FileUpload'
import { getFolderTitle } from '@/components/file/LazyFileTree'
import { DataTable } from '@/components/query-table'
import { DataTableColumnHeader } from '@/components/query-table/column-header'
import { DataTableToolbarConfig } from '@/components/query-table/toolbar'
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

import { IAccount, apiAdminAccountList } from '@/services/api/account'
import { AccessMode } from '@/services/api/auth'
import { FileItem, MoveFile, apiFileDelete, apiMkdir, apiMoveFile } from '@/services/api/file'
import { BaseUserInfo, apiGetBaseUserInfo } from '@/services/api/user'
import { apiXMLDownload } from '@/services/client'
import { IResponse } from '@/services/types'

import useIsAdmin from '@/hooks/use-admin'

import { FILE_SIZE_LIMITS, isFileSizeExceeded } from '@/utils/fileSize'
import { atomBreadcrumb, atomUserContext } from '@/utils/store'
import { showErrorToast } from '@/utils/toast'

import { cn } from '@/lib/utils'

import TooltipButton from '../button/tooltip-button'
import { Badge } from '../ui/badge'
import { FileDownloadProgress } from './FileDownloadProgress'
import { FileSelectDialog } from './FileSelectDialog'
import FolderNavigation from './FolderNavigation'

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
      placeholder: t('search.name.placeholder'),
      key: 'name',
    },
    filterOptions: [],
    getHeader: (key) => getHeader(key, t),
  }
}

interface SpacefileTableProps {
  apiGetFiles: (path: string) => Promise<IResponse<FileItem[] | undefined>>
  path: string
}

const FileActions = ({
  deleteFile,
  moveFile,
  isDir,
  name,
  path,
  canEdit,
  size,
}: {
  deleteFile: (path: string) => void
  moveFile: ({ fileData, path }: { fileData: MoveFile; path: string }) => void
  isDir: boolean
  name: string
  path: string
  canEdit: boolean
  size?: number
}) => {
  const [isMoveDialogOpen, setIsMoveDialogOpen] = useState(false)
  const [isDownloading, setIsDownloading] = useState(false)
  const [downloadProgress, setDownloadProgress] = useState(0)
  const [isProcessing, setIsProcessing] = useState(false)
  const { t } = useTranslation()

  const handleDownload = async () => {
    // 检查文件大小限制
    if (size && isFileSizeExceeded(size)) {
      toast.error(t('fileDownload.fileSizeError', { maxSize: FILE_SIZE_LIMITS.MAX_FILE_SIZE_TEXT }))
      return
    }

    setIsDownloading(true)
    setDownloadProgress(0)
    setIsProcessing(false)

    try {
      toast.info(t('fileDownload.inProgress'))
      const downloadPath = `ss/download${path ? (path.startsWith('/') ? path : '/' + path) : ''}/${name}`

      await apiXMLDownload(downloadPath, name, (progressEvent) => {
        const { loaded, total } = progressEvent
        const percentCompleted = Math.round((loaded * 100) / total)
        setDownloadProgress(percentCompleted)

        if (percentCompleted === 100) {
          setIsProcessing(true)
        }
      })

      toast.success(t('fileDownload.success'))
    } catch (error) {
      toast.error(
        t('fileDownload.error', {
          status: error instanceof Error ? error.message : '未知错误',
        })
      )
    } finally {
      setTimeout(() => {
        setIsDownloading(false)
        setIsProcessing(false)
        setDownloadProgress(0)
      }, 1000)
    }
  }
  return (
    <>
      <div className="flex flex-row items-center justify-end space-x-1">
        {/* 下载按钮（仅文件显示）*/}
        {!isDir ? (
          <TooltipButton
            variant="outline"
            className="size-8 p-0 hover:text-sky-700"
            tooltipContent={t('fileActions.download.tooltip')}
            disabled={isDownloading}
            onClick={handleDownload}
          >
            <DownloadIcon className="size-4" />
          </TooltipButton>
        ) : (
          <div className="size-8" />
        )}
        {/* 移动操作 （有读写权限显示）*/}
        {canEdit ? (
          <AlertDialog open={isMoveDialogOpen} onOpenChange={setIsMoveDialogOpen}>
            <AlertDialogTrigger asChild>
              <div>
                <TooltipButton
                  className="hover:text-highlight-orange h-8 w-8 p-0"
                  variant="outline"
                  size="icon"
                  tooltipContent={t('fileActions.move.tooltip', {
                    type: isDir ? t('fileActions.type.folder') : t('fileActions.type.file'),
                  })}
                >
                  {isDir ? (
                    <FolderOutputIcon size={16} strokeWidth={2} />
                  ) : (
                    <FileOutputIcon size={16} strokeWidth={2} />
                  )}
                </TooltipButton>
              </div>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle className="text-lg">
                  {t('fileActions.move.title', {
                    type: isDir ? t('fileActions.type.folder') : t('fileActions.type.file'),
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
        {canEdit ? (
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <div>
                <TooltipButton
                  className="hover:text-destructive h-8 w-8 p-0"
                  variant="outline"
                  size="icon"
                  tooltipContent={t('fileActions.delete.tooltip', {
                    type: isDir ? t('fileActions.type.folder') : t('fileActions.type.file'),
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
                    type: isDir ? t('fileActions.type.folder') : t('fileActions.type.file'),
                  })}
                </AlertDialogTitle>
                <AlertDialogDescription>
                  {t('fileActions.delete.description', {
                    type: isDir ? t('fileActions.type.folder') : t('fileActions.type.file'),
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

      {/* 下载进度显示 */}
      {(isDownloading || isProcessing) && (
        <FileDownloadProgress
          progress={downloadProgress}
          isProcessing={isProcessing}
          fileName={name}
        />
      )}
    </>
  )
}

export default function FileSystem({ apiGetFiles, path }: SpacefileTableProps) {
  const { t } = useTranslation()
  const { pathname } = useLocation()
  const navigate = useNavigate()
  const [dirName, setDirName] = useState<string>('')
  const setBreadcrumb = useSetAtom(atomBreadcrumb)
  const isAdmin = useIsAdmin()
  const router = useRouter()

  useEffect(() => {
    const baseURL = isAdmin ? '/admin/files' : '/portal/files'
    const pathParts = path.split('/').filter(Boolean)
    const breadcrumb = pathParts.map((value, index) => {
      value = decodeURIComponent(value)
      // 累加路径：取当前索引及之前的所有路径片段
      const cumulativePath = pathParts.slice(0, index + 1).join('/')
      return {
        label: getFolderTitle(t, value),
        href: `${baseURL}/${cumulativePath}`,
      }
    })
    setBreadcrumb([
      {
        label: t('navigation.fileManagement'),
        href: baseURL,
      },
      ...breadcrumb,
    ])

    return () => {
      setBreadcrumb([]) // 清空面包屑
    }
  }, [isAdmin, path, setBreadcrumb, t])

  const query = useQuery({
    queryKey: ['data', 'spacefile', path],
    queryFn: () => apiGetFiles(`${path}`),
    select: (res) => {
      return (
        res.data
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
    select: (res) => res.data,
    staleTime: 5 * 60 * 1000,
    enabled: isAdmin,
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
      return res.data
    },
    staleTime: 5 * 60 * 1000,
    enabled: isAdmin,
  })
  const accountMap = useMemo(() => {
    // 添加类型断言确保数据安全
    if (!accountInfoQuery.data || !Array.isArray(accountInfoQuery.data)) {
      return new Map<string, IAccount>()
    }
    return new Map(accountInfoQuery.data.map((a) => [a.space, a]))
  }, [accountInfoQuery.data])

  const isRoot = useMemo(() => {
    return path === ''
  }, [path])

  const isAdminUserSpace = useMemo(() => {
    const basePath = '/admin/files/spacefile/admin-user'
    return pathname === basePath
  }, [pathname])

  const isAdminAccountSpace = useMemo(() => {
    const basePath = '/admin/files/spacefile/admin-account'
    return pathname === basePath
  }, [pathname])

  const { mutate: deleteFile } = useMutation({
    mutationFn: (req: string) => apiFileDelete(req),
    onSuccess: async () => {
      toast.success(t('fileActions.delete.success'))
      await queryClient.invalidateQueries({
        queryKey: ['data', 'spacefile', path],
      })
    },
  })

  const token = useAtomValue(atomUserContext)
  const canEdit = useMemo(() => {
    if (path.startsWith('public')) {
      return token?.accessPublic === AccessMode.ReadWrite
    }
    if (path.startsWith('account')) {
      return token?.accessQueue === AccessMode.ReadWrite // 根据实际权限规则调整
    }
    if (path.startsWith('user') || path.startsWith('admin')) {
      return true
    }
    return false // 默认不显示
  }, [path, token?.accessPublic, token?.accessQueue])

  const { mutate: moveFile } = useMutation({
    mutationFn: ({ fileData, path }: { fileData: MoveFile; path: string }) =>
      apiMoveFile(fileData, path),
    onSuccess: async () => {
      toast.success(t('fileActions.move.success'))
      await queryClient.invalidateQueries({
        queryKey: ['data', 'spacefile', path],
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
          <div className="flex items-center justify-start gap-2">
            <Folder className="size-5 text-yellow-600 dark:text-yellow-400" />
            <TooltipButton
              tooltipContent="查看文件夹内容"
              onClick={() => navigate({ to: pathname + '/' + row.original.name })}
              variant="link"
              className="text-secondary-foreground h-8 px-0 text-left font-normal"
            >
              {row.getValue('name')}
            </TooltipButton>
          </div>
        ) : (
          <div className="flex items-center gap-2">
            <File className="text-muted-foreground size-5" />
            <span className="text-secondary-foreground">{row.getValue('name')}</span>
          </div>
        )
      }

      return (
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              {row.original.isdir ? (
                <div className="flex items-center">
                  <Folder className="mr-2 size-5 text-yellow-600 dark:text-yellow-400" />
                  <Button
                    onClick={() => navigate({ to: pathname + '/' + row.original.name })}
                    variant="link"
                    className="text-secondary-foreground hover:text-primary h-8 px-0 text-left font-normal"
                  >
                    {displayName}
                  </Button>
                </div>
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
        accessorKey: 'size',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('size', t)} />
        ),
        cell: ({ row }) => {
          if (row.original.isdir) {
            return (
              <>
                {row.original.size > 0 && (
                  <Badge variant="secondary">
                    {t('fileActions.size.tooltip', { size: row.original.size })}
                  </Badge>
                )}
              </>
            )
          } else {
            return (
              <Badge variant="outline">
                <FileSizeComponent size={row.getValue('size')} />
              </Badge>
            )
          }
        },
      },
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
        id: 'actions',
        enableHiding: false,
        cell: ({ row }) => {
          const { isdir, name, size } = row.original

          return (
            <FileActions
              deleteFile={deleteFile}
              moveFile={moveFile}
              isDir={isdir}
              name={name}
              path={path}
              canEdit={canEdit}
              size={size}
            />
          )
        },
      },
    ]
  }, [deleteFile, moveFile, path, canEdit, t])

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
        queryKey: ['data', 'spacefile', path],
      })
    },
  })

  return (
    <>
      {isRoot ? (
        <FolderNavigation data={query.data} isadmin={isAdmin} />
      ) : (
        <DataTable
          storageKey="spacefile"
          query={query}
          columns={columns}
          toolbarConfig={getToolbarConfig(t)}
        >
          <TooltipButton
            variant="outline"
            size="icon"
            onClick={() => {
              router.history.back()
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

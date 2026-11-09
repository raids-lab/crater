/**
 * Copyright 2026 The Crater Project Team, RAIDS-Lab
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
import { useQuery } from '@tanstack/react-query'
import { ArrowUpIcon, FolderOpenIcon, LoaderCircleIcon } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'

import { apiGetFiles } from '@/services/api/file'

import { getTensorboardPersonalStoragePath } from '@/utils/tensorboard'

interface PersonalDirectoryPickerProps {
  username: string
  value: string
  onSelect: (path: string) => void
}

export default function PersonalDirectoryPicker({
  username,
  value,
  onSelect,
}: PersonalDirectoryPickerProps) {
  const { t } = useTranslation()
  const personalRoot = `/home/${username}`
  const [open, setOpen] = useState(false)
  const [directory, setDirectory] = useState(personalRoot)
  const [search, setSearch] = useState('')
  const storagePath = getTensorboardPersonalStoragePath(directory, username)
  const { data, isFetching, isError, isSuccess, refetch } = useQuery({
    queryKey: ['tensorboard', 'personal-directory', username, storagePath],
    queryFn: () => apiGetFiles(storagePath ?? ''),
    enabled: open && storagePath !== null,
    retry: false,
  })
  const directories = (data?.data ?? [])
    .filter(
      (item) =>
        item.isdir &&
        item.name !== '.' &&
        item.name !== '..' &&
        !/[\\/]/.test(item.name) &&
        item.name.toLocaleLowerCase().includes(search.trim().toLocaleLowerCase())
    )
    .sort((left, right) => left.name.localeCompare(right.name))

  const navigate = (path: string) => {
    if (getTensorboardPersonalStoragePath(path, username) === null) return
    setDirectory(path)
    setSearch('')
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        if (nextOpen) {
          const initialPath = getTensorboardPersonalStoragePath(value, username)
          setDirectory(
            initialPath ? `${personalRoot}${initialPath.slice('user'.length)}` : personalRoot
          )
          setSearch('')
        }
        setOpen(nextOpen)
      }}
    >
      <DialogTrigger asChild>
        <Button type="button" variant="outline" className="w-fit gap-2">
          <FolderOpenIcon className="size-4" />
          {t('tensorboard.directory.browse')}
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('tensorboard.directory.title')}</DialogTitle>
          <DialogDescription>{t('tensorboard.directory.description')}</DialogDescription>
        </DialogHeader>
        <div className="flex items-center gap-2">
          <Button
            type="button"
            variant="outline"
            size="icon"
            disabled={directory === personalRoot}
            aria-label={t('tensorboard.directory.parent')}
            onClick={() => navigate(directory.slice(0, directory.lastIndexOf('/')))}
          >
            <ArrowUpIcon className="size-4" />
          </Button>
          <span className="min-w-0 flex-1 truncate font-mono text-sm" title={directory}>
            {directory}
          </span>
          <Button
            type="button"
            variant="outline"
            disabled={isFetching}
            onClick={() => navigate(personalRoot)}
          >
            {t('tensorboard.directory.root')}
          </Button>
        </div>
        <Input
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          placeholder={t('tensorboard.directory.search')}
          aria-label={t('tensorboard.directory.search')}
        />
        <div className="h-64 overflow-y-auto rounded-md border p-2" aria-busy={isFetching}>
          {isFetching ? (
            <p role="status" className="text-muted-foreground flex items-center gap-2 p-2 text-sm">
              <LoaderCircleIcon className="size-4 animate-spin" />
              {t('common.loading')}
            </p>
          ) : isError ? (
            <div role="alert" className="space-y-2 p-2">
              <p className="text-destructive text-sm">{t('tensorboard.directory.loadFailed')}</p>
              <Button type="button" variant="outline" onClick={() => refetch()}>
                {t('common.refresh')}
              </Button>
            </div>
          ) : directories.length === 0 ? (
            <p className="text-muted-foreground p-2 text-sm">{t('tensorboard.directory.empty')}</p>
          ) : (
            directories.map((item) => (
              <Button
                key={item.name}
                type="button"
                variant="ghost"
                className="w-full justify-start gap-2"
                onClick={() => navigate(`${directory}/${item.name}`)}
              >
                <FolderOpenIcon className="size-4 shrink-0" />
                <span className="truncate">{item.name}</span>
              </Button>
            ))
          )}
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => setOpen(false)}>
            {t('common.cancel')}
          </Button>
          <Button
            type="button"
            disabled={!isSuccess || isFetching || storagePath === null}
            onClick={() => {
              onSelect(directory)
              setOpen(false)
            }}
          >
            {t('tensorboard.directory.select')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

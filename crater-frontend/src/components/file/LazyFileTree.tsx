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
import { useTranslation } from 'react-i18next'
import React, { useEffect, useMemo } from 'react'
import * as AccordionPrimitive from '@radix-ui/react-accordion'
import { ScrollArea, ScrollBar } from '@/components/ui/scroll-area'
import { cn } from '@/lib/utils'
import { ChevronRight, FileDigitIcon, FolderIcon, type LucideIcon } from 'lucide-react'
import useResizeObserver from 'use-resize-observer'
import { useMutation, useQuery } from '@tanstack/react-query'
import { FileItem, apiGetFiles, apiGetRWFiles } from '@/services/api/file'

interface TreeDataItem {
  id: string
  name: string
  icon: LucideIcon
  isdir: boolean
  hasChildren: boolean
}

export const getFolderTitle = (t: (key: string) => string, folder: string) => {
  if (folder === 'public') {
    return t('tree.folderTitle.public')
  } else if (folder === 'account') {
    return t('tree.folderTitle.account')
  }
  return t('tree.folderTitle.default')
}

export const getAdminFolderTitle = (t: (key: string) => string, folder: string) => {
  if (folder === 'admin-public') {
    return t('tree.adminFolderTitle.public')
  } else if (folder === 'admin-account') {
    return t('tree.adminFolderTitle.account')
  }
  return t('tree.adminFolderTitle.default')
}

type TreeProps = React.HTMLAttributes<HTMLDivElement> & {
  initialSlelectedItemId?: string
  onSelectChange?: (item: TreeDataItem | undefined) => void
  className?: string
  isrw?: boolean
}

const Tree = ({
  ref,
  initialSlelectedItemId,
  onSelectChange,
  isrw,
  className,
  ...props
}: TreeProps & {
  ref?: React.RefObject<HTMLDivElement>
}) => {
  const [selectedItemId, setSelectedItemId] = React.useState<string | undefined>(
    initialSlelectedItemId
  )

  const handleSelectChange = React.useCallback(
    (item: TreeDataItem | undefined) => {
      setSelectedItemId(item?.id)
      if (onSelectChange) {
        onSelectChange(item)
      }
    },
    [onSelectChange]
  )

  const { data } = useQuery({
    queryKey: ['directory', 'list', isrw],
    queryFn: () => (isrw ? apiGetRWFiles('') : apiGetFiles('')),
    select: (res) =>
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
          return a.name.localeCompare(b.name)
        }) ?? [],
  })

  const { ref: refRoot, width, height } = useResizeObserver()

  return (
    <div ref={refRoot} className={cn('overflow-hidden', className)}>
      <ScrollArea style={{ width, height }}>
        <div className="relative p-2" {...props}>
          <ul>
            {data?.map((item, index) => (
              <TreeItem
                level={0}
                key={index}
                currentPath={item.name}
                data={item}
                ref={ref}
                isrw={isrw}
                selectedItemId={selectedItemId}
                handleSelectChange={handleSelectChange}
              />
            ))}
          </ul>
        </div>
        <ScrollBar orientation="horizontal" />
      </ScrollArea>
    </div>
  )
}
Tree.displayName = 'Tree'

type TreeItemProps = TreeProps & {
  data: FileItem
  level: number
  currentPath: string
  selectedItemId?: string
  handleSelectChange: (item: TreeDataItem | undefined) => void
}

const TreeItem = ({
  ref,
  level,
  data,
  currentPath,
  isrw,
  selectedItemId,
  handleSelectChange,
  className,
  ...props
}: TreeItemProps & {
  ref?: React.RefObject<HTMLDivElement>
}) => {
  const { t } = useTranslation()

  const item: TreeDataItem = useMemo(() => {
    return {
      id: currentPath,
      name: level === 0 ? getFolderTitle(t, data.name) : data.name,
      icon: data.isdir ? FolderIcon : FileDigitIcon,
      isdir: data.isdir,
      hasChildren: data.isdir && data.size > 0,
    }
  }, [currentPath, data, level, t])

  useEffect(() => {
    if (data.isdir && data.size > 0) {
      // create default childrens with length of data.size
      const newChildren: FileItem[] = Array.from({ length: data.size }, () => ({
        isdir: false,
        name: '',
        size: 0,
        modifytime: '',
      }))
      setChildren(newChildren)
    }
  }, [data])

  const [children, setChildren] = React.useState<FileItem[]>([])
  const [childrenInitialized, setChildrenInitialized] = React.useState(false)

  const { mutate: getChildren } = useMutation({
    mutationFn: () => (isrw ? apiGetRWFiles(currentPath) : apiGetFiles(currentPath)),
    onSuccess: (fileList) => {
      const children =
        fileList.data.data?.sort((a, b) => {
          if (a.isdir && !b.isdir) {
            return -1 // a在b之前
          } else if (!a.isdir && b.isdir) {
            return 1 // a在b之后
          } else {
            return a.name.localeCompare(b.name)
          }
        }) ?? []
      if (children) {
        setChildren(children)
        setChildrenInitialized(true)
      }
    },
  })

  return (
    <li>
      <div ref={ref} role="tree" className={className} {...props}>
        {item.hasChildren ? (
          <AccordionPrimitive.Root type="multiple">
            <AccordionPrimitive.Item value={item.id}>
              <AccordionTrigger
                className={cn(
                  'before:bg-muted/80 px-2 before:absolute before:left-0 before:-z-10 before:h-[1.75rem] before:w-full before:opacity-0 hover:before:opacity-100',
                  selectedItemId === item.id &&
                    'text-accent-foreground before:border-l-accent-foreground/50 before:bg-accent before:border-l-2 before:opacity-100 dark:before:border-0'
                )}
                onClick={() => {
                  if (!childrenInitialized) {
                    getChildren()
                  }
                  handleSelectChange(item)
                }}
              >
                <item.icon
                  className="text-accent-foreground/50 mr-2 size-4 shrink-0"
                  aria-hidden="true"
                />
                <span className="truncate text-sm">{item.name}</span>
              </AccordionTrigger>
              <AccordionContent className="pl-6">
                <ul>
                  {children.map((data, index) => {
                    return (
                      <TreeItem
                        level={level + 1}
                        key={index}
                        data={data}
                        isrw={isrw}
                        currentPath={`${currentPath}/${data.name}`}
                        selectedItemId={selectedItemId}
                        handleSelectChange={handleSelectChange}
                      />
                    )
                  })}
                </ul>
              </AccordionContent>
            </AccordionPrimitive.Item>
          </AccordionPrimitive.Root>
        ) : (
          <Leaf
            item={item}
            isSelected={selectedItemId === item.id}
            onClick={() => handleSelectChange(item)}
          />
        )}
      </div>
    </li>
  )
}
TreeItem.displayName = 'TreeItem'

const Leaf = ({
  ref,
  className,
  item,
  isSelected,
  ...props
}: React.HTMLAttributes<HTMLDivElement> & {
  ref?: React.RefObject<HTMLDivElement>
  item: TreeDataItem
  isSelected?: boolean
}) => {
  return (
    <div
      ref={ref}
      className={cn(
        'before:bg-muted/80 flex cursor-pointer items-center px-2 py-2 before:absolute before:right-1 before:left-0 before:-z-10 before:h-[1.75rem] before:w-full before:opacity-0 hover:before:opacity-100',
        className,
        isSelected &&
          'text-accent-foreground before:border-l-accent-foreground/50 before:bg-accent before:border-l-2 before:opacity-100 dark:before:border-0'
      )}
      {...props}
    >
      <item.icon className="text-accent-foreground/50 mr-2 size-4 shrink-0" aria-hidden="true" />
      <span className="grow truncate text-sm">{item.name}</span>
    </div>
  )
}
Leaf.displayName = 'Leaf'

const AccordionTrigger = ({
  ref,
  className,
  children,
  ...props
}: React.ComponentProps<typeof AccordionPrimitive.Trigger>) => (
  <AccordionPrimitive.Header>
    <AccordionPrimitive.Trigger
      ref={ref}
      className={cn(
        'flex w-full flex-1 items-center py-2 transition-all [&[data-state=open]>svg]:last:rotate-90',
        className
      )}
      {...props}
    >
      {children}
      <ChevronRight className="text-accent-foreground/50 ml-auto size-4 shrink-0 transition-transform duration-200" />
    </AccordionPrimitive.Trigger>
  </AccordionPrimitive.Header>
)
AccordionTrigger.displayName = AccordionPrimitive.Trigger.displayName

const AccordionContent = ({
  ref,
  className,
  children,
  ...props
}: React.ComponentProps<typeof AccordionPrimitive.Content>) => (
  <AccordionPrimitive.Content
    ref={ref}
    className={cn(
      'data-[state=closed]:animate-accordion-up data-[state=open]:animate-accordion-down overflow-hidden text-sm transition-all',
      className
    )}
    {...props}
  >
    <div className="pt-0 pb-0">{children}</div>
  </AccordionPrimitive.Content>
)
AccordionContent.displayName = AccordionPrimitive.Content.displayName

export { Tree, type TreeDataItem }

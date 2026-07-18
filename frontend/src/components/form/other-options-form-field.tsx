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
import { useQuery } from '@tanstack/react-query'
import { CheckIcon, ChevronsUpDownIcon, CircleHelpIcon, SettingsIcon, XIcon } from 'lucide-react'
import { useState } from 'react'
import { FieldPath, FieldValues, PathValue, UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command'
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'

import { apiGetNodes } from '@/services/api/cluster'

import { NodeSelectorMode } from '@/utils/form'

import { cn } from '@/lib/utils'

import AccordionCard from './accordion-card'

export function getOtherCardTitle(t: (key: string) => string) {
  return t('otherOptionsFormCard.accordionTitle')
}

// useNodeNames returns the schedulable node names for the cluster, used to
// power the node include / exclude selectors.
function useNodeNames() {
  return useQuery({
    queryKey: ['cluster', 'nodes', 'brief'],
    queryFn: () => apiGetNodes(),
    select: (res) => res.data.map((n) => n.name).sort((a, b) => a.localeCompare(b)),
  })
}

// NodeMultiSelect is a searchable multi-select for node scheduling constraints.
function NodeMultiSelect({
  values,
  onChange,
  options,
  placeholder,
  selectedLabel,
  emptyLabel,
  removeLabel,
}: {
  values: string[]
  onChange: (values: string[]) => void
  options: string[]
  placeholder: string
  selectedLabel: string
  emptyLabel: string
  removeLabel: (name: string) => string
}) {
  const [open, setOpen] = useState(false)
  const toggle = (name: string) => {
    if (values.includes(name)) {
      onChange(values.filter((v) => v !== name))
    } else {
      onChange([...values, name])
    }
  }
  return (
    <div className="space-y-2">
      <Popover modal open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <FormControl>
            <Button
              variant="outline"
              role="combobox"
              type="button"
              aria-expanded={open}
              className={cn(
                'w-full justify-between font-normal',
                values.length === 0 && 'text-muted-foreground'
              )}
            >
              <span className="truncate">{values.length > 0 ? selectedLabel : placeholder}</span>
              <ChevronsUpDownIcon className="ml-2 size-4 shrink-0 opacity-50" />
            </Button>
          </FormControl>
        </PopoverTrigger>
        <PopoverContent
          className="p-0"
          style={{ width: 'var(--radix-popover-trigger-width)' }}
          align="start"
        >
          <Command>
            <CommandInput placeholder={placeholder} className="h-9" />
            <CommandList>
              <CommandEmpty>{emptyLabel}</CommandEmpty>
              <CommandGroup>
                {options.map((name) => {
                  const selected = values.includes(name)
                  return (
                    <CommandItem
                      key={name}
                      value={name}
                      onSelect={() => toggle(name)}
                      className="font-mono"
                    >
                      <div
                        className={cn(
                          'border-primary mr-2 flex size-4 items-center justify-center rounded-sm border',
                          selected ? 'bg-primary text-primary-foreground' : 'opacity-50'
                        )}
                      >
                        {selected && <CheckIcon className="size-3" />}
                      </div>
                      {name}
                    </CommandItem>
                  )
                })}
              </CommandGroup>
            </CommandList>
          </Command>
        </PopoverContent>
      </Popover>
      {values.length > 0 && (
        <div className="flex flex-wrap gap-1">
          {values.map((name) => (
            <Badge key={name} variant="secondary" className="gap-1 font-mono font-normal">
              {name}
              <button
                type="button"
                onClick={() => toggle(name)}
                className="hover:text-destructive"
                aria-label={removeLabel(name)}
              >
                <XIcon className="size-3" />
              </button>
            </Badge>
          ))}
        </div>
      )}
    </div>
  )
}

interface OtherOptionsFormCardProps<T extends FieldValues> {
  form: UseFormReturn<T>
  alertEnabledPath: FieldPath<T>
  nodeSelectorEnablePath: FieldPath<T>
  nodeSelectorModePath: FieldPath<T>
  nodeSelectorNodesPath: FieldPath<T>
  cpuPinningEnabledPath?: FieldPath<T> // 可选的 CPU 绑核路径
  open: boolean
  setOpen: (open: boolean) => void
}

export function OtherOptionsFormCard<T extends FieldValues>({
  form,
  alertEnabledPath,
  nodeSelectorEnablePath,
  nodeSelectorModePath,
  nodeSelectorNodesPath,
  cpuPinningEnabledPath,
  open,
  setOpen,
}: OtherOptionsFormCardProps<T>) {
  const { t } = useTranslation()
  const nodeSelectorEnabled = form.watch(nodeSelectorEnablePath)
  const watchedNodeSelectorMode = form.watch(nodeSelectorModePath) as NodeSelectorMode | undefined
  const nodeSelectorMode =
    watchedNodeSelectorMode === NodeSelectorMode.Exclude
      ? NodeSelectorMode.Exclude
      : NodeSelectorMode.Include
  const nodeSelectorIsIncludeMode = nodeSelectorMode === NodeSelectorMode.Include
  const [cpuPinningConfirmOpen, setCpuPinningConfirmOpen] = useState(false)
  const { data: nodeNames = [] } = useNodeNames()

  return (
    <>
      <AccordionCard
        cardTitle={t('otherOptionsFormCard.accordionTitle')}
        icon={SettingsIcon}
        open={open}
        setOpen={setOpen}
      >
        <div className="mt-3 space-y-3">
          <FormField
            control={form.control}
            name={alertEnabledPath}
            render={({ field }) => (
              <FormItem className="flex flex-row items-center justify-between space-y-0 space-x-0">
                <FormLabel className="font-normal">
                  {t('otherOptionsFormCard.receiveStatusNotifications')}
                  <TooltipProvider delayDuration={100}>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <CircleHelpIcon className="text-muted-foreground size-4 hover:cursor-help" />
                      </TooltipTrigger>
                      <TooltipContent>
                        <p className="mb-0.5 font-semibold">
                          {t('otherOptionsFormCard.tooltip.receiveEmailNotifications')}
                        </p>
                        <p>{t('otherOptionsFormCard.tooltip.notification1')}</p>
                        <p>{t('otherOptionsFormCard.tooltip.notification2')}</p>
                        <p>{t('otherOptionsFormCard.tooltip.notification3')}</p>
                        <p>{t('otherOptionsFormCard.tooltip.notification4')}</p>
                        <p>{t('otherOptionsFormCard.tooltip.notification5')}</p>
                        <p>{t('otherOptionsFormCard.tooltip.emailSupport')}</p>
                      </TooltipContent>
                    </Tooltip>
                  </TooltipProvider>
                </FormLabel>
                <FormControl>
                  <Switch checked={field.value} onCheckedChange={field.onChange} />
                </FormControl>
              </FormItem>
            )}
          />
          {cpuPinningEnabledPath && (
            <FormField
              control={form.control}
              name={cpuPinningEnabledPath}
              render={({ field }) => (
                <>
                  <FormItem className="flex flex-row items-center justify-between space-y-0 space-x-0">
                    <FormLabel className="font-normal">
                      {t('otherOptionsFormCard.enableCpuPinning')}
                      <TooltipProvider delayDuration={100}>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <CircleHelpIcon className="text-muted-foreground size-4 hover:cursor-help" />
                          </TooltipTrigger>
                          <TooltipContent>
                            {t('otherOptionsFormCard.tooltip.cpuPinning')}
                          </TooltipContent>
                        </Tooltip>
                      </TooltipProvider>
                    </FormLabel>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        onCheckedChange={(checked) => {
                          if (checked && !field.value) {
                            setCpuPinningConfirmOpen(true)
                            return
                          }

                          field.onChange(checked)
                        }}
                      />
                    </FormControl>
                  </FormItem>
                  <AlertDialog open={cpuPinningConfirmOpen} onOpenChange={setCpuPinningConfirmOpen}>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>
                          {t('otherOptionsFormCard.cpuPinningConfirm.title')}
                        </AlertDialogTitle>
                        <AlertDialogDescription>
                          {t('otherOptionsFormCard.cpuPinningConfirm.description')}
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel>
                          {t('otherOptionsFormCard.cpuPinningConfirm.cancel')}
                        </AlertDialogCancel>
                        <AlertDialogAction
                          onClick={() => {
                            field.onChange(true)
                          }}
                        >
                          {t('otherOptionsFormCard.cpuPinningConfirm.confirm')}
                        </AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                </>
              )}
            />
          )}
          <div className="space-y-1.5">
            <FormField
              control={form.control}
              name={nodeSelectorEnablePath}
              render={({ field }) => (
                <FormItem className="flex flex-row items-center justify-between space-y-0 space-x-0 pb-1.5">
                  <FormLabel className="font-normal">
                    {t('otherOptionsFormCard.targetNodeControl')}
                    <TooltipProvider delayDuration={100}>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <CircleHelpIcon className="text-muted-foreground size-4 hover:cursor-help" />
                        </TooltipTrigger>
                        <TooltipContent>
                          {t('otherOptionsFormCard.tooltip.targetNodeControl')}
                        </TooltipContent>
                      </Tooltip>
                    </TooltipProvider>
                  </FormLabel>
                  <FormControl>
                    <Switch checked={field.value} onCheckedChange={field.onChange} />
                  </FormControl>
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name={nodeSelectorModePath}
              render={({ field }) => (
                <FormItem
                  className={cn('gap-1', {
                    hidden: !nodeSelectorEnabled,
                  })}
                >
                  <FormControl>
                    <Tabs
                      value={nodeSelectorMode}
                      onValueChange={(value) => {
                        field.onChange(value as NodeSelectorMode)
                      }}
                      className="w-full"
                    >
                      <TabsList className="grid w-full grid-cols-2">
                        <TabsTrigger value={NodeSelectorMode.Include} className="cursor-pointer">
                          {t('otherOptionsFormCard.mode.include')}
                        </TabsTrigger>
                        <TabsTrigger value={NodeSelectorMode.Exclude} className="cursor-pointer">
                          {t('otherOptionsFormCard.mode.exclude')}
                        </TabsTrigger>
                      </TabsList>
                    </Tabs>
                  </FormControl>
                  <FormDescription>
                    {nodeSelectorIsIncludeMode
                      ? t('otherOptionsFormCard.includeModeDescription')
                      : t('otherOptionsFormCard.excludeModeDescription')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name={nodeSelectorNodesPath}
              render={({ field }) => (
                <FormItem
                  className={cn({
                    hidden: !nodeSelectorEnabled,
                  })}
                >
                  <NodeMultiSelect
                    values={(field.value as string[]) ?? []}
                    onChange={(values) => field.onChange(values as PathValue<T, FieldPath<T>>)}
                    options={nodeNames}
                    placeholder={
                      nodeSelectorIsIncludeMode
                        ? t('otherOptionsFormCard.includeNodePlaceholder')
                        : t('otherOptionsFormCard.excludeNodePlaceholder')
                    }
                    selectedLabel={t('otherOptionsFormCard.selectedNodeCount', {
                      count: ((field.value as string[]) ?? []).length,
                    })}
                    emptyLabel={t('otherOptionsFormCard.emptyNodeList')}
                    removeLabel={(name) => t('otherOptionsFormCard.removeNodeAriaLabel', { name })}
                  />
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
        </div>
      </AccordionCard>
    </>
  )
}

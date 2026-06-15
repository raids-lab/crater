import { BotIcon, Loader2Icon, RotateCcwIcon, SaveIcon } from 'lucide-react'
import { UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'

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
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import {
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

export const createStorageDecisionSettingsSchema = (t: (key: string) => string) =>
  z
    .object({
      decisionMode: z.enum(['agent', 'direct']),
      configSource: z.enum(['platform', 'custom']),
      baseUrl: z.string().optional(),
      modelName: z.string().optional(),
      apiKey: z.string().optional(),
    })
    .superRefine((values, ctx) => {
      if (values.configSource !== 'custom') {
        return
      }

      if (!values.baseUrl || !/^https?:\/\//.test(values.baseUrl)) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['baseUrl'],
          message: t('systemConfig.llm.validation.url'),
        })
      }

      if (!values.modelName) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['modelName'],
          message: t('systemConfig.llm.validation.model'),
        })
      }
    })

export type StorageDecisionFormSchema = z.infer<
  ReturnType<typeof createStorageDecisionSettingsSchema>
>

interface StorageDecisionSettingsProps {
  form: UseFormReturn<StorageDecisionFormSchema>
  isPending: boolean
  onSubmit: (values: StorageDecisionFormSchema, validate: boolean) => void
  onReset: () => void
}

export function StorageDecisionSettings({
  form,
  isPending,
  onSubmit,
  onReset,
}: StorageDecisionSettingsProps) {
  const { t } = useTranslation()
  const decisionMode = form.watch('decisionMode')
  const configSource = form.watch('configSource')

  return (
    <>
      <CardHeader>
        <div className="flex items-center gap-2">
          <BotIcon className="text-primary h-5 w-5" />
          <CardTitle>{t('systemConfig.storageDecision.title')}</CardTitle>
        </div>
        <CardDescription>{t('systemConfig.storageDecision.description')}</CardDescription>
      </CardHeader>
      <Form {...form}>
        <form>
          <CardContent className="space-y-4">
            <FormField
              control={form.control}
              name="decisionMode"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('systemConfig.llm.decisionMode')}</FormLabel>
                  <Select onValueChange={field.onChange} value={field.value}>
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectItem value="agent">{t('systemConfig.llm.mode.agent')}</SelectItem>
                      <SelectItem value="direct">{t('systemConfig.llm.mode.direct')}</SelectItem>
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="configSource"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('systemConfig.storageDecision.configSource')}</FormLabel>
                  <Select onValueChange={field.onChange} value={field.value}>
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectItem value="platform">
                        {t('systemConfig.storageDecision.source.platform')}
                      </SelectItem>
                      <SelectItem value="custom">
                        {t('systemConfig.storageDecision.source.custom')}
                      </SelectItem>
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />

            {configSource === 'platform' && (
              <p className="text-muted-foreground rounded-md border px-3 py-2 text-sm">
                {t('systemConfig.storageDecision.platformHint')}
              </p>
            )}

            {configSource === 'custom' && (
              <>
                <FormField
                  control={form.control}
                  name="baseUrl"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('systemConfig.llm.baseUrl')}</FormLabel>
                      <FormControl>
                        <Input placeholder="http://192.168.5.68:30186/v1" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="modelName"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('systemConfig.llm.modelName')}</FormLabel>
                      <FormControl>
                        <Input placeholder="storage-governance-qwen25-7b" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="apiKey"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('systemConfig.llm.apiKey')}</FormLabel>
                      <FormControl>
                        <Input type="password" placeholder="crater-local / sk-..." {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </>
            )}
          </CardContent>
          <CardFooter className="bg-muted/10 flex flex-wrap justify-between gap-4 px-6 py-4">
            <p className="text-muted-foreground text-xs italic">
              {decisionMode === 'direct'
                ? t('systemConfig.storageDecision.directTip')
                : t('systemConfig.storageDecision.tip')}
            </p>
            <div className="flex gap-2">
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <Button type="button" variant="destructive" disabled={isPending}>
                    <RotateCcwIcon className="mr-2 h-4 w-4" />
                    {t('common.reset')}
                  </Button>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>
                      {t('systemConfig.storageDecision.resetConfirmTitle')}
                    </AlertDialogTitle>
                    <AlertDialogDescription>
                      {t('systemConfig.storageDecision.resetConfirmDesc')}
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>{t('common.cancel')}</AlertDialogCancel>
                    <AlertDialogAction onClick={onReset}>{t('common.confirm')}</AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>

              <Button
                type="button"
                onClick={form.handleSubmit((values) => onSubmit(values, true))}
                disabled={isPending}
              >
                {isPending ? (
                  <Loader2Icon className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <SaveIcon className="mr-2 h-4 w-4" />
                )}
                {t('common.save')}
              </Button>
            </div>
          </CardFooter>
        </form>
      </Form>
    </>
  )
}

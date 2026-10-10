import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useAtom } from 'jotai'
import { BotIcon, CheckCircle2Icon, KeyRoundIcon, Trash2Icon } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
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
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'

import {
  apiDeletePersonalLLMConfig,
  apiGetPersonalLLMConfig,
  apiTestPersonalLLMConfig,
  apiUpdatePersonalLLMConfig,
} from '@/services/api/agent'

import { atomAgentUserFloatingButtonEnabled } from '@/utils/store'

const maskedAPIKey = '********************************************'

const formSchema = z.object({
  baseUrl: z.string().url().startsWith('https://'),
  modelName: z.string().trim().min(1).max(256),
  apiKey: z.string().trim().min(1).max(8192),
})

type FormValues = z.infer<typeof formSchema>

export function PersonalLLMSettings() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [floatingButtonEnabled, setFloatingButtonEnabled] = useAtom(
    atomAgentUserFloatingButtonEnabled
  )
  const queryKey = ['agent', 'personal-llm-config']
  const initializedFromServer = useRef(false)
  const [pendingConfig, setPendingConfig] = useState<FormValues>()
  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: { baseUrl: '', modelName: '', apiKey: '' },
  })
  const { data } = useQuery({
    queryKey,
    queryFn: apiGetPersonalLLMConfig,
    select: (response) => response.data,
  })

  useEffect(() => {
    if (!data || initializedFromServer.current) return
    initializedFromServer.current = true
    form.reset(
      data.personalConfigured
        ? { baseUrl: data.baseUrl, modelName: data.modelName, apiKey: data.apiKey }
        : { baseUrl: '', modelName: '', apiKey: '' }
    )
  }, [data, form])

  const saveMutation = useMutation({
    mutationFn: (values: FormValues) => apiUpdatePersonalLLMConfig({ ...values, validate: true }),
    onSuccess: (_response, values) => {
      toast.success(t('agentSettings.saveSuccess'))
      form.reset({ baseUrl: values.baseUrl, modelName: values.modelName, apiKey: maskedAPIKey })
      queryClient.invalidateQueries({ queryKey })
    },
  })
  const testMutation = useMutation({
    mutationFn: (values: FormValues) => apiTestPersonalLLMConfig(values),
    onSuccess: () => toast.success(t('agentSettings.testSuccess')),
  })
  const deleteMutation = useMutation({
    mutationFn: apiDeletePersonalLLMConfig,
    onSuccess: () => {
      toast.success(t('agentSettings.deleteSuccess'))
      form.reset({ baseUrl: '', modelName: '', apiKey: '' })
      queryClient.invalidateQueries({ queryKey })
    },
  })

  const sourceText = data?.personalConfigured
    ? t('agentSettings.sourcePersonal')
    : data?.platformConfigured
      ? t('agentSettings.sourcePlatform')
      : t('agentSettings.sourceMissing')

  return (
    <Card>
      <CardHeader>
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="space-y-1">
            <CardTitle className="flex items-center gap-2">
              <BotIcon className="text-primary size-5" />
              {t('agentSettings.title')}
            </CardTitle>
            <CardDescription>{t('agentSettings.description')}</CardDescription>
          </div>
          <Badge variant={data?.effectiveSource ? 'secondary' : 'destructive'}>{sourceText}</Badge>
        </div>
      </CardHeader>
      <CardContent>
        <div className="mb-6 flex items-center justify-between gap-4 rounded-lg border p-4">
          <div className="space-y-1">
            <p className="text-sm font-medium">{t('agentSettings.floatingButton')}</p>
            <p className="text-muted-foreground text-sm">
              {t('agentSettings.floatingButtonDescription')}
            </p>
          </div>
          <Switch
            checked={floatingButtonEnabled}
            aria-label={t('agentSettings.floatingButton')}
            onCheckedChange={setFloatingButtonEnabled}
          />
        </div>
        <Form {...form}>
          <form
            className="space-y-5"
            autoComplete="off"
            onSubmit={form.handleSubmit(setPendingConfig)}
          >
            <FormField
              control={form.control}
              name="baseUrl"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('agentSettings.baseUrl')}</FormLabel>
                  <FormControl>
                    <Input
                      {...field}
                      className="font-mono"
                      autoComplete="off"
                      placeholder="https://api.example.com/v1"
                    />
                  </FormControl>
                  <FormDescription>{t('agentSettings.baseUrlDescription')}</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="modelName"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('agentSettings.modelName')}</FormLabel>
                  <FormControl>
                    <Input
                      {...field}
                      className="font-mono"
                      autoComplete="off"
                      placeholder="model-name"
                    />
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
                  <FormLabel>{t('agentSettings.apiKey')}</FormLabel>
                  <FormControl>
                    <Input
                      {...field}
                      className="font-mono"
                      type="password"
                      autoComplete="new-password"
                    />
                  </FormControl>
                  <FormDescription>{t('agentSettings.apiKeyDescription')}</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <div className="flex flex-wrap items-center gap-3">
              <Button type="submit" disabled={saveMutation.isPending}>
                <KeyRoundIcon />
                {saveMutation.isPending ? t('agentSettings.verifying') : t('agentSettings.save')}
              </Button>
              <Button
                type="button"
                variant="secondary"
                disabled={testMutation.isPending}
                onClick={form.handleSubmit((values) => testMutation.mutate(values))}
              >
                <CheckCircle2Icon />
                {testMutation.isPending ? t('agentSettings.testing') : t('agentSettings.test')}
              </Button>
              {data?.personalConfigured && (
                <AlertDialog>
                  <AlertDialogTrigger asChild>
                    <Button type="button" variant="outline" disabled={deleteMutation.isPending}>
                      <Trash2Icon />
                      {t('agentSettings.delete')}
                    </Button>
                  </AlertDialogTrigger>
                  <AlertDialogContent>
                    <AlertDialogHeader>
                      <AlertDialogTitle>{t('agentSettings.deleteTitle')}</AlertDialogTitle>
                      <AlertDialogDescription>
                        {t('agentSettings.deleteDescription')}
                      </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                      <AlertDialogCancel>{t('agentSettings.cancel')}</AlertDialogCancel>
                      <AlertDialogAction onClick={() => deleteMutation.mutate()}>
                        {t('agentSettings.confirmDelete')}
                      </AlertDialogAction>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                </AlertDialog>
              )}
            </div>
          </form>
        </Form>
        <AlertDialog
          open={pendingConfig !== undefined}
          onOpenChange={(open) => {
            if (!open && !saveMutation.isPending) setPendingConfig(undefined)
          }}
        >
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>{t('agentSettings.saveTitle')}</AlertDialogTitle>
              <AlertDialogDescription>{t('agentSettings.saveDescription')}</AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel disabled={saveMutation.isPending}>
                {t('agentSettings.cancel')}
              </AlertDialogCancel>
              <AlertDialogAction
                disabled={saveMutation.isPending}
                onClick={() => {
                  if (!pendingConfig) return
                  saveMutation.mutate(pendingConfig, {
                    onSuccess: () => setPendingConfig(undefined),
                  })
                }}
              >
                {saveMutation.isPending
                  ? t('agentSettings.verifying')
                  : t('agentSettings.confirmSave')}
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </CardContent>
    </Card>
  )
}

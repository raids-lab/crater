import { useQuery } from '@tanstack/react-query'
import {
  CircleHelpIcon,
  DownloadIcon,
  GaugeIcon,
  Loader2Icon,
  SaveIcon,
  ShieldCheckIcon,
  UsersIcon,
} from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import SelectBox from '@/components/custom/select-box'
import SimpleTooltip from '@/components/label/simple-tooltip'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui-custom/alert-dialog'

import { apiAdminUserList } from '@/services/api/admin/user'
import { IAdminModelDownloadLimitConfig } from '@/services/api/system-config'

interface ModelDownloadLimitSettingsProps {
  config?: IAdminModelDownloadLimitConfig
  isPending: boolean
  isLoading: boolean
  isError: boolean
  onRetry: () => void
  onSubmit: (config: IAdminModelDownloadLimitConfig) => Promise<unknown>
}

export function ModelDownloadLimitSettings({
  config,
  isPending,
  isLoading,
  isError,
  onRetry,
  onSubmit,
}: ModelDownloadLimitSettingsProps) {
  const { t } = useTranslation()
  const [enabled, setEnabled] = useState(true)
  const [maxConcurrent, setMaxConcurrent] = useState('5')
  const [windowHours, setWindowHours] = useState('2')
  const [maxSuccessfulDownloads, setMaxSuccessfulDownloads] = useState('5')
  const [whitelistUserIds, setWhitelistUserIds] = useState<string[]>([])
  const [pendingConfig, setPendingConfig] = useState<IAdminModelDownloadLimitConfig | null>(null)
  const [isConfirming, setIsConfirming] = useState(false)

  const { data: userOptions = [] } = useQuery({
    queryKey: ['admin', 'userlist'],
    queryFn: apiAdminUserList,
    select: (res) =>
      res.data.map((user) => ({
        value: String(user.id),
        label: user.attributes.nickname || user.name,
        labelNote: user.name,
      })),
  })

  useEffect(() => {
    if (!config) return
    setEnabled(config.enabled)
    setMaxConcurrent(String(config.maxConcurrent))
    setWindowHours(String(config.windowHours))
    setMaxSuccessfulDownloads(String(config.maxSuccessfulDownloads))
    setWhitelistUserIds(config.whitelistUserIds.map(String))
  }, [config])

  const handleSubmit = () => {
    const values = [maxConcurrent, windowHours, maxSuccessfulDownloads].map(Number)
    if (values.some((value) => !Number.isInteger(value) || value <= 0)) {
      toast.error(t('systemConfig.modelDownloadLimit.invalidPositiveInteger'))
      return
    }
    setPendingConfig({
      enabled,
      maxConcurrent: values[0],
      windowHours: values[1],
      maxSuccessfulDownloads: values[2],
      whitelistUserIds: whitelistUserIds.map(Number),
    })
  }

  const controlsDisabled = isPending || isLoading || isError || !config
  const confirmationPending = isPending || isConfirming

  return (
    <>
      <CardHeader>
        <div className="flex items-center gap-2">
          <DownloadIcon
            className={enabled ? 'h-5 w-5 text-blue-500' : 'text-muted-foreground h-5 w-5'}
          />
          <CardTitle>{t('systemConfig.modelDownloadLimit.title')}</CardTitle>
        </div>
        <CardDescription>{t('systemConfig.modelDownloadLimit.description')}</CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading && (
          <div
            className="bg-muted/50 text-muted-foreground mb-5 flex items-center gap-2 rounded-md border p-3 text-sm"
            role="status"
          >
            <Loader2Icon className="size-4 animate-spin" />
            {t('systemConfig.modelDownloadLimit.loading')}
          </div>
        )}
        {isError && (
          <div
            className="border-destructive/50 bg-destructive/5 text-destructive mb-5 flex items-center justify-between gap-3 rounded-md border p-3 text-sm"
            role="alert"
          >
            <span>{t('systemConfig.modelDownloadLimit.loadError')}</span>
            <Button type="button" variant="outline" size="sm" onClick={onRetry}>
              {t('systemConfig.modelDownloadLimit.retryLoad')}
            </Button>
          </div>
        )}
        <Tabs defaultValue="rules" className="gap-5">
          <TabsList className="grid w-full max-w-md grid-cols-2">
            <TabsTrigger value="rules">
              <GaugeIcon />
              {t('systemConfig.modelDownloadLimit.rulesTab')}
            </TabsTrigger>
            <TabsTrigger value="whitelist">
              <ShieldCheckIcon />
              {t('systemConfig.modelDownloadLimit.whitelistTab')}
              {whitelistUserIds.length > 0 && (
                <Badge variant="secondary" className="ml-1 h-5 min-w-5 px-1.5">
                  {whitelistUserIds.length}
                </Badge>
              )}
            </TabsTrigger>
          </TabsList>

          <TabsContent value="rules" className="mt-0 space-y-4">
            <div className="flex items-center justify-between rounded-lg border p-4 shadow-sm">
              <div>
                <Label className="text-base">{t('systemConfig.modelDownloadLimit.enabled')}</Label>
                <p className="text-muted-foreground text-sm">
                  {t('systemConfig.modelDownloadLimit.scopeDescription')}
                </p>
              </div>
              <Switch checked={enabled} onCheckedChange={setEnabled} disabled={controlsDisabled} />
            </div>
            <div className="grid gap-4 md:grid-cols-3">
              <div className="space-y-2">
                <Label htmlFor="model-download-max-concurrent">
                  {t('systemConfig.modelDownloadLimit.maxConcurrent')}
                  <SimpleTooltip
                    tooltip={t('systemConfig.modelDownloadLimit.maxConcurrentTooltip')}
                  >
                    <CircleHelpIcon className="text-muted-foreground ml-1 inline size-4 cursor-help" />
                  </SimpleTooltip>
                </Label>
                <Input
                  id="model-download-max-concurrent"
                  type="number"
                  min={1}
                  value={maxConcurrent}
                  onChange={(event) => setMaxConcurrent(event.target.value)}
                  disabled={controlsDisabled}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="model-download-window-hours">
                  {t('systemConfig.modelDownloadLimit.windowHours')}
                  <SimpleTooltip tooltip={t('systemConfig.modelDownloadLimit.windowHoursTooltip')}>
                    <CircleHelpIcon className="text-muted-foreground ml-1 inline size-4 cursor-help" />
                  </SimpleTooltip>
                </Label>
                <Input
                  id="model-download-window-hours"
                  type="number"
                  min={1}
                  value={windowHours}
                  onChange={(event) => setWindowHours(event.target.value)}
                  disabled={controlsDisabled}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="model-download-max-successful-downloads">
                  {t('systemConfig.modelDownloadLimit.maxSuccessfulDownloads')}
                  <SimpleTooltip
                    tooltip={t('systemConfig.modelDownloadLimit.maxSuccessfulDownloadsTooltip')}
                  >
                    <CircleHelpIcon className="text-muted-foreground ml-1 inline size-4 cursor-help" />
                  </SimpleTooltip>
                </Label>
                <Input
                  id="model-download-max-successful-downloads"
                  type="number"
                  min={1}
                  value={maxSuccessfulDownloads}
                  onChange={(event) => setMaxSuccessfulDownloads(event.target.value)}
                  disabled={controlsDisabled}
                />
              </div>
            </div>
          </TabsContent>

          <TabsContent value="whitelist" className="mt-0">
            <div className="from-muted/60 to-background rounded-xl border bg-gradient-to-br p-5">
              <div className="mb-4 flex items-start justify-between gap-4">
                <div className="flex gap-3">
                  <div className="bg-primary/10 text-primary flex size-10 shrink-0 items-center justify-center rounded-lg">
                    <UsersIcon className="size-5" />
                  </div>
                  <div>
                    <Label className="text-base">
                      {t('systemConfig.modelDownloadLimit.whitelistTitle')}
                    </Label>
                    <p className="text-muted-foreground mt-1 text-sm">
                      {t('systemConfig.modelDownloadLimit.whitelistDescription')}
                    </p>
                  </div>
                </div>
                <Badge variant="outline" className="shrink-0">
                  {t('systemConfig.modelDownloadLimit.whitelistCount', {
                    count: whitelistUserIds.length,
                  })}
                </Badge>
              </div>
              <SelectBox
                className="bg-background min-h-10"
                options={userOptions}
                value={whitelistUserIds}
                onChange={setWhitelistUserIds}
                withDialogOverlay={false}
                disabled={controlsDisabled}
                placeholder={t('systemConfig.modelDownloadLimit.whitelistPlaceholder')}
                inputPlaceholder={t('systemConfig.modelDownloadLimit.whitelistSearchPlaceholder')}
              />
            </div>
          </TabsContent>
        </Tabs>
      </CardContent>
      <CardFooter className="flex justify-end border-t px-6 py-4">
        <Button onClick={handleSubmit} disabled={controlsDisabled}>
          {isPending ? (
            <Loader2Icon className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <SaveIcon className="mr-2 h-4 w-4" />
          )}
          {t('common.save')}
        </Button>
      </CardFooter>

      <AlertDialog
        open={pendingConfig !== null}
        onOpenChange={(open) => !open && !confirmationPending && setPendingConfig(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('systemConfig.modelDownloadLimit.confirmTitle')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('systemConfig.modelDownloadLimit.confirmDescription')}
            </AlertDialogDescription>
          </AlertDialogHeader>
          {pendingConfig && (
            <dl className="bg-muted/50 grid grid-cols-[minmax(8rem,auto)_1fr] gap-x-4 gap-y-2 rounded-md p-4 text-sm">
              <dt className="text-muted-foreground">
                {t('systemConfig.modelDownloadLimit.confirmEnabled')}
              </dt>
              <dd>
                {pendingConfig.enabled
                  ? t('systemConfig.modelDownloadLimit.statusEnabled')
                  : t('systemConfig.modelDownloadLimit.statusDisabled')}
              </dd>
              <dt className="text-muted-foreground">
                {t('systemConfig.modelDownloadLimit.maxConcurrent')}
              </dt>
              <dd>{pendingConfig.maxConcurrent}</dd>
              <dt className="text-muted-foreground">
                {t('systemConfig.modelDownloadLimit.windowHours')}
              </dt>
              <dd>{pendingConfig.windowHours}</dd>
              <dt className="text-muted-foreground">
                {t('systemConfig.modelDownloadLimit.maxSuccessfulDownloads')}
              </dt>
              <dd>{pendingConfig.maxSuccessfulDownloads}</dd>
              <dt className="text-muted-foreground">
                {t('systemConfig.modelDownloadLimit.confirmWhitelistCount')}
              </dt>
              <dd>{pendingConfig.whitelistUserIds.length}</dd>
            </dl>
          )}
          <p className="text-muted-foreground text-sm">
            {t('systemConfig.modelDownloadLimit.confirmScope')}
          </p>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={confirmationPending}>
              {t('common.cancel')}
            </AlertDialogCancel>
            <AlertDialogAction
              disabled={confirmationPending || pendingConfig === null}
              onClick={(event) => {
                event.preventDefault()
                if (!pendingConfig || confirmationPending) return
                setIsConfirming(true)
                void onSubmit(pendingConfig)
                  .then(() => setPendingConfig(null))
                  .catch(() => undefined)
                  .finally(() => setIsConfirming(false))
              }}
            >
              {confirmationPending && <Loader2Icon className="mr-2 size-4 animate-spin" />}
              {confirmationPending
                ? t('systemConfig.modelDownloadLimit.saving')
                : t('systemConfig.modelDownloadLimit.confirmSave')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}

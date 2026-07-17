import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Loader2Icon, SaveIcon, ShieldBanIcon } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'

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

import {
  IUserBanPolicy,
  apiAdminGetUserBanPolicy,
  apiAdminUpdateUserBanPolicy,
} from '@/services/api/system-config'

import { showErrorToast } from '@/utils/toast'

const DEFAULT_POLICY: IUserBanPolicy = {
  allowPlatformAccess: false,
  allowJobSubmission: false,
  allowImageBuild: false,
  allowModelDownload: false,
  allowDatasetDownload: false,
}

const policyFields: Array<{
  key: keyof IUserBanPolicy
  labelKey: string
  descriptionKey: string
}> = [
  {
    key: 'allowPlatformAccess',
    labelKey: 'systemConfig.userBan.allowPlatformAccess',
    descriptionKey: 'systemConfig.userBan.allowPlatformAccessDesc',
  },
  {
    key: 'allowJobSubmission',
    labelKey: 'systemConfig.userBan.allowJobSubmission',
    descriptionKey: 'systemConfig.userBan.allowJobSubmissionDesc',
  },
  {
    key: 'allowImageBuild',
    labelKey: 'systemConfig.userBan.allowImageBuild',
    descriptionKey: 'systemConfig.userBan.allowImageBuildDesc',
  },
  {
    key: 'allowModelDownload',
    labelKey: 'systemConfig.userBan.allowModelDownload',
    descriptionKey: 'systemConfig.userBan.allowModelDownloadDesc',
  },
  {
    key: 'allowDatasetDownload',
    labelKey: 'systemConfig.userBan.allowDatasetDownload',
    descriptionKey: 'systemConfig.userBan.allowDatasetDownloadDesc',
  },
]

export function UserBanSettings() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [policy, setPolicy] = useState<IUserBanPolicy>(DEFAULT_POLICY)

  const policyQuery = useQuery({
    queryKey: ['admin', 'system-config', 'user-ban'],
    queryFn: () => apiAdminGetUserBanPolicy().then((res) => res.data),
  })

  useEffect(() => {
    if (policyQuery.data) setPolicy(policyQuery.data)
  }, [policyQuery.data])

  const updateMutation = useMutation({
    mutationFn: apiAdminUpdateUserBanPolicy,
    onSuccess: async (data) => {
      setPolicy(data.data)
      await queryClient.invalidateQueries({ queryKey: ['admin', 'system-config', 'user-ban'] })
      toast.success(t('systemConfig.userBan.saveSuccess'))
    },
    onError: showErrorToast,
  })

  return (
    <>
      <CardHeader>
        <div className="flex items-center gap-2">
          <ShieldBanIcon className="text-destructive size-5" />
          <CardTitle>{t('systemConfig.userBan.title')}</CardTitle>
        </div>
        <CardDescription>{t('systemConfig.userBan.description')}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {policyQuery.isError && (
          <div className="flex flex-col items-start gap-3 rounded-lg border p-4 sm:flex-row sm:items-center sm:justify-between">
            <p className="text-destructive text-sm">{t('systemConfig.userBan.loadError')}</p>
            <Button variant="outline" size="sm" onClick={() => void policyQuery.refetch()}>
              {t('common.refresh')}
            </Button>
          </div>
        )}
        <div className="grid gap-4 md:grid-cols-2">
          {policyFields.map((field) => (
            <div
              key={field.key}
              className="flex min-w-0 items-center justify-between gap-4 rounded-lg border p-4"
            >
              <div className="min-w-0 space-y-1">
                <Label className="text-sm font-medium sm:text-base">{t(field.labelKey)}</Label>
                <p className="text-muted-foreground text-xs leading-5 sm:text-sm">
                  {t(field.descriptionKey)}
                </p>
              </div>
              <Switch
                className="shrink-0"
                checked={policy[field.key]}
                disabled={policyQuery.isLoading || policyQuery.isError || updateMutation.isPending}
                onCheckedChange={(checked) =>
                  setPolicy((current) => ({ ...current, [field.key]: checked }))
                }
              />
            </div>
          ))}
        </div>
        {!policy.allowPlatformAccess && (
          <p className="text-muted-foreground text-sm">
            {t('systemConfig.userBan.platformDisabledHint')}
          </p>
        )}
        <div className="flex justify-end">
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button
                disabled={policyQuery.isLoading || policyQuery.isError || updateMutation.isPending}
              >
                {updateMutation.isPending ? (
                  <Loader2Icon className="mr-2 size-4 animate-spin" />
                ) : (
                  <SaveIcon className="mr-2 size-4" />
                )}
                {updateMutation.isPending ? t('common.saving') : t('common.saveChanges')}
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>{t('systemConfig.userBan.confirmTitle')}</AlertDialogTitle>
                <AlertDialogDescription>
                  {t('systemConfig.userBan.confirmDescription')}
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>{t('common.cancel')}</AlertDialogCancel>
                <AlertDialogAction
                  disabled={updateMutation.isPending}
                  onClick={() => updateMutation.mutate(policy)}
                >
                  {t('common.confirm')}
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </div>
      </CardContent>
    </>
  )
}

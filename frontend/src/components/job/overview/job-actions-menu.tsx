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
import { DurationDialog } from '@/routes/admin/jobs/-components/duration-dialog'
import { Link, linkOptions } from '@tanstack/react-router'
import {
  EllipsisVerticalIcon as DotsHorizontalIcon,
  InfoIcon,
  LockIcon,
  RedoDotIcon,
  SquareIcon,
  Trash2Icon,
  UnlockIcon,
  XIcon,
} from 'lucide-react'
import { useCallback, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

import { getNewJobLink } from '@/components/job/new-job-button'
import { JobLockMenuItem, JobLockSheet } from '@/components/job/overview/job-lock-sheet'
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

import { IJobInfo, JobStatus, getJobStateType } from '@/services/api/vcjob'

// Link Options for portal job navigation
const portalJobDetailLinkOptions = linkOptions({
  to: '/portal/jobs/detail/$name',
  params: { name: '' },
  search: { tab: '' },
})

// Link Options for admin job navigation
const adminJobDetailLinkOptions = linkOptions({
  to: '/admin/jobs/$name',
  params: { name: '' },
  search: { tab: '' },
})

interface JobActionsMenuProps {
  jobInfo: IJobInfo
  onDelete: (jobName: string) => void
  isAdminView?: boolean
  onLockSuccess?: () => void
}

export const JobActionsMenu = ({
  jobInfo,
  onDelete,
  isAdminView = false,
  onLockSuccess,
}: JobActionsMenuProps) => {
  const { t } = useTranslation()
  const [isLockSheetOpen, setIsLockSheetOpen] = useState(false)
  const [isLockDialogOpen, setIsLockDialogOpen] = useState(false)
  const [isExtendDialogOpen, setIsExtendDialogOpen] = useState(false)
  const jobStatus = getJobStateType(jobInfo.status)
  const option = useMemo(() => {
    return getNewJobLink(jobInfo.jobType)
  }, [jobInfo.jobType])

  // 暂时隐藏申请锁定功能，工单审批功能不完善
  const canExtend = true // jobStatus === JobStatus.Running

  // Admin view: handle lock/unlock
  const handleLockClick = useCallback(() => {
    setIsLockDialogOpen(true)
  }, [])

  const handleExtendClick = useCallback(() => {
    setIsExtendDialogOpen(true)
  }, [])

  const handleLockSuccess = useCallback(() => {
    setIsLockDialogOpen(false)
    setIsExtendDialogOpen(false)
    onLockSuccess?.()
  }, [onLockSuccess])

  // Determine detail link based on view
  const detailLinkOptions = isAdminView ? adminJobDetailLinkOptions : portalJobDetailLinkOptions
  const shouldStop = jobInfo.status !== 'Deleted' && jobInfo.status !== 'Freed'

  // Admin view: only show lock button for Running or Pending (NotStarted) jobs
  const isPending = jobInfo.status === 'Pending' || jobStatus === JobStatus.NotStarted
  const canLockInAdminView = isAdminView && (jobStatus === JobStatus.Running || isPending)

  return (
    <>
      <AlertDialog>
        <JobLockSheet
          isOpen={isLockSheetOpen}
          onOpenChange={setIsLockSheetOpen}
          jobName={jobInfo.jobName}
        />
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" className="h-8 w-8 p-0">
              <span className="sr-only">{t('jobs.actions.dropdown.ariaLabel')}</span>
              <DotsHorizontalIcon className="size-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuLabel className="text-muted-foreground text-xs">
              {t('jobs.actions.dropdown.title')}
            </DropdownMenuLabel>
            <DropdownMenuItem asChild>
              <Link {...detailLinkOptions} params={{ name: jobInfo.jobName }}>
                <InfoIcon className="text-highlight-emerald size-4" />
                {isAdminView
                  ? t('adminJobOverview.actions.dropdown.details')
                  : t('jobs.actions.dropdown.details')}
              </Link>
            </DropdownMenuItem>
            {!isAdminView && (
              <>
                <DropdownMenuItem asChild>
                  <Link {...option} search={{ fromJob: jobInfo.jobName }}>
                    <RedoDotIcon className="text-highlight-purple size-4" />
                    {t('jobs.actions.dropdown.clone')}
                  </Link>
                </DropdownMenuItem>
                {canExtend && jobStatus === JobStatus.Running && (
                  <JobLockMenuItem jobInfo={jobInfo} onLock={() => setIsLockSheetOpen(true)} />
                )}
              </>
            )}
            {isAdminView && canLockInAdminView && (
              <>
                <DropdownMenuItem
                  onClick={handleLockClick}
                  title={t('adminJobOverview.actions.dropdown.lockTitle')}
                >
                  {jobInfo.locked ? (
                    <UnlockIcon className="text-highlight-purple size-4" />
                  ) : (
                    <LockIcon className="text-highlight-purple size-4" />
                  )}
                  {jobInfo.locked
                    ? t('adminJobOverview.actions.dropdown.unlock')
                    : t('adminJobOverview.actions.dropdown.lock')}
                </DropdownMenuItem>
                {jobInfo.locked && (
                  <DropdownMenuItem
                    onClick={handleExtendClick}
                    title={t('adminJobOverview.actions.dropdown.lockTitle')}
                  >
                    <LockIcon className="text-highlight-purple size-4" />
                    {t('adminJobOverview.actions.dropdown.extend')}
                  </DropdownMenuItem>
                )}
              </>
            )}
            <AlertDialogTrigger asChild>
              <DropdownMenuItem className="group">
                {isAdminView ? (
                  shouldStop ? (
                    <SquareIcon className="text-highlight-orange size-4" />
                  ) : (
                    <Trash2Icon className="text-destructive size-4" />
                  )
                ) : jobStatus === JobStatus.NotStarted ? (
                  <XIcon className="text-highlight-orange size-4" />
                ) : jobStatus === JobStatus.Running ? (
                  <SquareIcon className="text-highlight-orange size-4" />
                ) : (
                  <Trash2Icon className="text-destructive size-4" />
                )}
                {isAdminView
                  ? shouldStop
                    ? t('adminJobOverview.actions.dropdown.stop')
                    : t('adminJobOverview.actions.dropdown.delete')
                  : jobStatus === JobStatus.NotStarted
                    ? t('jobs.actions.dropdown.cancel')
                    : jobStatus === JobStatus.Running
                      ? t('jobs.actions.dropdown.stop')
                      : t('jobs.actions.dropdown.delete')}
              </DropdownMenuItem>
            </AlertDialogTrigger>
          </DropdownMenuContent>
        </DropdownMenu>

        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {isAdminView
                ? shouldStop
                  ? t('adminJobOverview.dialog.stop.title')
                  : t('adminJobOverview.dialog.delete.title')
                : jobStatus === JobStatus.NotStarted
                  ? t('jobs.dialog.cancel.title')
                  : jobStatus === JobStatus.Running
                    ? t('jobs.dialog.stop.title')
                    : t('jobs.dialog.delete.title')}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {isAdminView
                ? shouldStop
                  ? t('adminJobOverview.dialog.stop.description', { name: jobInfo?.name })
                  : t('adminJobOverview.dialog.delete.description', { name: jobInfo?.name })
                : jobStatus === JobStatus.NotStarted
                  ? t('jobs.dialog.cancel.description', { name: jobInfo.name })
                  : jobStatus === JobStatus.Running
                    ? t('jobs.dialog.stop.description', { name: jobInfo.name })
                    : t('jobs.dialog.delete.description', { name: jobInfo.name })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('jobs.actions.dropdown.cancel')}</AlertDialogCancel>
            <AlertDialogAction variant="destructive" onClick={() => onDelete(jobInfo.jobName)}>
              {isAdminView
                ? shouldStop
                  ? t('adminJobOverview.dialog.stop.action')
                  : t('adminJobOverview.dialog.delete.action')
                : jobStatus === JobStatus.NotStarted
                  ? t('jobs.dialog.cancel.action')
                  : jobStatus === JobStatus.Running
                    ? t('jobs.actions.dropdown.stop')
                    : t('jobs.actions.dropdown.delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
      {isAdminView && (
        <>
          <DurationDialog
            jobs={[jobInfo]}
            open={isLockDialogOpen}
            setOpen={setIsLockDialogOpen}
            onSuccess={handleLockSuccess}
          />
          <DurationDialog
            jobs={[jobInfo]}
            open={isExtendDialogOpen}
            setOpen={setIsExtendDialogOpen}
            onSuccess={handleLockSuccess}
            setExtend={true}
          />
        </>
      )}
    </>
  )
}

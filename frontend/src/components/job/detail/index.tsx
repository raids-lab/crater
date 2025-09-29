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
import { useMutation, useQuery } from '@tanstack/react-query'
import { Link, useNavigate, useRouter } from '@tanstack/react-router'
import { useAtomValue } from 'jotai'
import {
  ActivityIcon,
  BarChartBigIcon,
  CalendarIcon,
  CheckCircleIcon,
  ClipboardListIcon,
  ClockIcon,
  CreditCardIcon,
  FileSlidersIcon,
  GaugeIcon,
  GpuIcon,
  HistoryIcon,
  LayoutGridIcon,
  RedoDotIcon,
  SquareIcon,
  Trash2Icon,
  UserRoundIcon,
  XIcon,
} from 'lucide-react'
import { useMemo } from 'react'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'

import JobPhaseLabel from '@/components/badge/job-phase-badge'
import JobTypeLabel from '@/components/badge/job-type-badge'
import TooltipButton from '@/components/button/tooltip-button'
import { CodeContent } from '@/components/codeblock/config-dialog'
import { LazyContent } from '@/components/codeblock/dialog'
import { TimeDistance } from '@/components/custom/time-distance'
import JupyterIcon from '@/components/icon/jupyter-icon'
import PrefixLinkButton from '@/components/job/detail/job-link-button'
import SimpleTooltip from '@/components/label/simple-tooltip'
import UserLabel from '@/components/label/user-label'
import DetailPage from '@/components/layout/detail-page'
import { DetailPageCoreProps } from '@/components/layout/detail-page'
import GrafanaIframe from '@/components/layout/embed/grafana-iframe'
import PageTitle from '@/components/layout/page-title'
import { EventTimeline } from '@/components/layout/timeline/event-timeline'
import ProfileDashboard from '@/components/metrics/profile-dashboard'
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

import { apiGetPodIngresses } from '@/services/api/tool'
import {
  JobPhase,
  JobStatus,
  JobType,
  apiJobDelete,
  apiJobGetDetail,
  apiJobGetEvent,
  apiJobGetPods,
  apiJobGetYaml,
  getJobStateType,
} from '@/services/api/vcjob'

import useFixedLayout from '@/hooks/use-fixed-layout'

import { hasNvidiaGPU } from '@/utils/resource'
import { configGrafanaJobAtom } from '@/utils/store/config'
import { getDaysDifference } from '@/utils/time'

import { REFETCH_INTERVAL } from '@/lib/constants'

import { getNewJobLink } from '../new-job-button'
import JobOrderList from './job-order-list'
import { PodTable } from './pod-table'
import { SSHPortDialog } from './s-s-h-port-dialog'

export default function BaseCore({ jobName, ...props }: DetailPageCoreProps & { jobName: string }) {
  useFixedLayout()
  const navigate = useNavigate()
  const router = useRouter()
  const grafanaJob = useAtomValue(configGrafanaJobAtom)

  // 获取作业详情
  const { data, isLoading } = useQuery({
    queryKey: ['job', 'detail', jobName],
    queryFn: () => apiJobGetDetail(jobName),
    select: (res) => res.data,
    refetchInterval: REFETCH_INTERVAL,
  })

  // Pod 相关信息获取
  const podQuery = useQuery({
    queryKey: ['job', 'detail', jobName, 'pods'],
    queryFn: () => apiJobGetPods(jobName),
    select: (res) => res.data.sort((a, b) => a.name.localeCompare(b.name)),
    enabled: !!jobName,
  })

  // 使用 useMemo 缓存 namespace 和 podName，避免每次渲染都重新取值
  const [namespace, podName] = useMemo(() => {
    const pod = podQuery.data?.[0]
    return [pod?.namespace, pod?.name]
  }, [podQuery.data])

  // ingress 查询依赖 namespace 和 podName，且 enabled 依赖 podQuery.isSuccess
  const { data: ingressList = [] } = useQuery({
    queryKey: ['ingresses', namespace, podName],
    queryFn: async () => {
      if (!namespace || !podName) return []
      const response = await apiGetPodIngresses(namespace, podName)
      return response.data.ingresses
    },
    enabled: !!namespace && !!podName && podQuery.isSuccess,
  })

  // 用 useMemo 缓存 ingressNames 和 ingressPrefixes，避免不必要的计算
  const ingressNames = useMemo(
    () => (Array.isArray(ingressList) ? ingressList.map((ing: { name: string }) => ing.name) : []),
    [ingressList]
  )
  const ingressPrefixes = useMemo(
    () =>
      Array.isArray(ingressList) ? ingressList.map((ing: { prefix: string }) => ing.prefix) : [],
    [ingressList]
  )

  const { mutate: deleteJTask } = useMutation({
    mutationFn: () => apiJobDelete(jobName),
    onSuccess: () => {
      toast.success('作业已删除')
      router.history.back()
    },
  })

  const showGPUDashboard = useMemo(() => {
    if (!data) {
      return false
    }
    return hasNvidiaGPU(data.resources)
  }, [data])

  const isCompletedOver3Days = useMemo(() => {
    return getDaysDifference(data?.completedAt) > 3
  }, [data?.completedAt])

  const jobStatus = useMemo(() => {
    if (!data) {
      return JobStatus.Unknown
    }
    return getJobStateType(data.status)
  }, [data])

  if (isLoading || !data) {
    return <></>
  }

  const fromTime = data.startedAt ? new Date(data.startedAt).toISOString() : 'now-3h'
  const toTime = data.completedAt ? new Date(data.completedAt).toISOString() : 'now'

  return (
    <DetailPage
      {...props}
      header={
        <PageTitle
          title={data.name}
          tipComponent={<JobTypeLabel jobType={data.jobType} />}
          description={data.jobName}
          descriptionCopiable
        >
          <div className="flex flex-row gap-3">
            {(data.jobType === JobType.Jupyter || data.jobType === JobType.Custom) &&
              data.status === JobPhase.Running && (
                <PrefixLinkButton names={ingressNames} prefixes={ingressPrefixes} />
              )}
            {(data.jobType === JobType.Jupyter || data.jobType === JobType.Custom) &&
              data.status === JobPhase.Running && (
                <SSHPortDialog jobName={jobName} userName={data.username} />
              )}
            {data.jobType === JobType.Jupyter && data.status === JobPhase.Running && (
              <SimpleTooltip tooltip="打开 Jupyter Lab">
                <Button variant="secondary" className="cursor-pointer" asChild>
                  <Link to="/ingress/jupyter/$name" params={{ name: data.jobName }} target="_blank">
                    <JupyterIcon className="size-4" />
                    Jupyter Lab
                  </Link>
                </Button>
              </SimpleTooltip>
            )}
            <AlertDialog>
              <AlertDialogTrigger asChild>
                {jobStatus === JobStatus.NotStarted ? (
                  <Button
                    title="取消作业"
                    className="bg-highlight-orange hover:bg-highlight-orange/90 cursor-pointer"
                  >
                    <XIcon className="size-4" />
                    取消作业
                  </Button>
                ) : jobStatus === JobStatus.Running ? (
                  <Button
                    title="停止作业"
                    className="bg-highlight-orange hover:bg-highlight-orange/90 cursor-pointer"
                  >
                    <SquareIcon className="size-4" />
                    停止作业
                  </Button>
                ) : (
                  <Button variant="destructive" title="删除作业" className="cursor-pointer">
                    <Trash2Icon className="size-4" />
                    删除作业
                  </Button>
                )}
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>
                    {jobStatus === JobStatus.NotStarted
                      ? '取消作业'
                      : jobStatus === JobStatus.Running
                        ? '停止作业'
                        : '删除作业'}
                  </AlertDialogTitle>
                  <AlertDialogDescription>
                    作业 {data.name} 将
                    {jobStatus === JobStatus.NotStarted
                      ? '取消，是否放弃排队？'
                      : jobStatus === JobStatus.Running
                        ? '停止，请确认已经保存好所需数据。'
                        : '删除，所有数据将被清理。'}
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>取消</AlertDialogCancel>
                  <AlertDialogAction
                    variant="destructive"
                    onClick={() => {
                      deleteJTask()
                    }}
                  >
                    {jobStatus === JobStatus.NotStarted
                      ? '确认'
                      : jobStatus === JobStatus.Running
                        ? '停止'
                        : '删除'}
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </div>
        </PageTitle>
      }
      info={[
        { title: '账户', icon: CreditCardIcon, value: data.queue },
        {
          title: '用户',
          icon: UserRoundIcon,
          value: <UserLabel info={data.userInfo} />,
        },
        {
          title: '状态',
          icon: ActivityIcon,
          value: <JobPhaseLabel jobPhase={data.status} />,
        },
        {
          title: '创建于',
          icon: CalendarIcon,
          value: <TimeDistance date={data.createdAt} />,
        },
        {
          title: '开始于',
          icon: ClockIcon,
          value: <TimeDistance date={data.startedAt} />,
        },
        {
          title: '完成于',
          icon: CheckCircleIcon,
          value: <TimeDistance date={data.completedAt} />,
        },
      ]}
      tabs={[
        {
          key: 'profile',
          icon: BarChartBigIcon,
          label: '统计信息',
          children: <ProfileDashboard data={data} />,
          scrollable: true,
          hidden:
            !data.profileData ||
            jobStatus === JobStatus.Running ||
            jobStatus === JobStatus.NotStarted,
        },
        {
          key: 'base',
          icon: LayoutGridIcon,
          label: '基本信息',
          children: <PodTable jobName={jobName} userName={data.username} />,
          scrollable: true,
          hidden: jobStatus === JobStatus.MetadataOnly || isCompletedOver3Days,
        },
        {
          key: 'config',
          icon: FileSlidersIcon,
          label: '作业配置',
          children: (
            <LazyContent
              name={jobName}
              type="yaml"
              fetchData={apiJobGetYaml}
              renderData={(yaml) => (
                <CodeContent
                  data={yaml}
                  language={'yaml'}
                  moreActions={
                    <>
                      <TooltipButton
                        size="icon"
                        variant="outline"
                        className="size-8"
                        tooltipContent="克隆作业"
                        onClick={() => {
                          const option = getNewJobLink(data.jobType)
                          navigate({
                            ...option,
                            search: { fromJob: data.jobName, fromTemplate: 0 },
                          })
                        }}
                      >
                        <RedoDotIcon />
                      </TooltipButton>
                    </>
                  }
                />
              )}
            />
          ),
        },
        {
          key: 'event',
          icon: HistoryIcon,
          label: jobStatus === JobStatus.Terminated ? '历史事件' : '作业事件',
          children: (
            <LazyContent
              name={jobName}
              type="event"
              fetchData={apiJobGetEvent}
              renderData={(events) => (
                <>
                  {events.length > 0 ? (
                    <EventTimeline items={events} />
                  ) : (
                    <EventTimeline items={data.events ?? []} />
                  )}
                </>
              )}
            />
          ),
          scrollable: true,
        },
        {
          key: 'order',
          icon: ClipboardListIcon,
          label: '相关工单',
          children: <JobOrderList jobName={data.jobName} />,
          scrollable: true,
        },
        {
          key: 'monitor',
          icon: GaugeIcon,
          label: '基础监控',
          children: (
            <GrafanaIframe
              baseSrc={`${grafanaJob.basic}?var-job=${data.jobName}&from=${fromTime}&to=${toTime}`}
            />
          ),
          hidden: jobStatus === JobStatus.NotStarted,
        },
        {
          key: 'gpu',
          icon: GpuIcon,
          label: '加速卡监控',
          children: (
            <GrafanaIframe
              baseSrc={`${grafanaJob.nvidia}?var-job=${data.jobName}&from=${fromTime}&to=${toTime}`}
            />
          ),
          hidden: !showGPUDashboard || jobStatus === JobStatus.NotStarted,
        },
      ]}
    />
  )
}

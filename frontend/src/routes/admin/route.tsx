import { useQuery } from '@tanstack/react-query'
// [新增]
import { Outlet, createFileRoute, redirect, useLocation } from '@tanstack/react-router'
import {
  AlarmClockIcon,
  BarChartBigIcon,
  BoxIcon,
  ClipboardCheckIcon,
  DatabaseIcon,
  FlaskConicalIcon,
  FolderIcon,
  GpuIcon,
  LayoutDashboard,
  ServerIcon,
  SettingsIcon,
  UserRoundIcon,
  UsersRoundIcon,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import AppLayout from '@/components/layout/app-layout'
import { NavGroupProps } from '@/components/sidebar/types'

import { Role } from '@/services/api/auth'
import { apiAdminGetGpuAnalysisStatus } from '@/services/api/system-config'

export const Route = createFileRoute('/admin')({
  beforeLoad: ({ context, location }) => {
    if (!context.auth.isAuthenticated || context.auth.context?.rolePlatform !== Role.Admin) {
      throw redirect({
        to: '/auth',
        search: {
          // Save current location for redirect after login
          redirect: location.href,
          token: '',
        },
      })
    }
  },
  component: RouteComponent,
})

const useAdminSidebarGroups = (): NavGroupProps[] => {
  const { t } = useTranslation()

  // [新增] 获取 GPU 分析功能的开启状态
  const { data: gpuStatus } = useQuery({
    queryKey: ['admin', 'system-config', 'gpu-status'],
    queryFn: () => apiAdminGetGpuAnalysisStatus().then((res) => res.data),
    staleTime: 1000 * 60 * 5, // 建议设置缓存时间，避免每次点击侧边栏都请求
  })

  // 判断是否开启
  const showGpuAnalysis = gpuStatus?.enabled ?? false

  return [
    {
      title: t('sidebar.resourceAndMonitoring'),
      items: [
        {
          title: t('navigation.resourceManagement'),
          icon: ServerIcon,
          items: [
            {
              title: t('navigation.nodeManagement'),
              url: '/admin/cluster/nodes',
            },
            {
              title: t('navigation.resourceManagement'),
              url: '/admin/cluster/resources',
            },
          ],
        },
        {
          title: t('navigation.clusterMonitoring'),
          icon: BarChartBigIcon,
          items: [
            {
              title: t('navigation.gpuMonitoring'),
              url: '/admin/monitor/gpu',
            },
            {
              title: t('navigation.freeResources'),
              url: '/admin/monitor/idle',
            },
            {
              title: t('navigation.networkMonitoring'),
              url: '/admin/monitor/network',
            },
          ],
        },
        {
          title: t('navigation.platformStatistics', { defaultValue: 'Platform Stats' }),
          url: '/admin/statistics', // 指向刚才创建的路由
          icon: LayoutDashboard,
        },
      ],
    },
    {
      title: t('sidebar.jobsAndServices'),
      items: [
        {
          title: t('navigation.jobManagement'),
          url: '/admin/jobs',
          icon: FlaskConicalIcon,
        },
        {
          title: t('navigation.cronPolicy'),
          url: '/admin/cronjobs',
          icon: AlarmClockIcon,
        },
        // [修改] 条件渲染：只有开启时才把该对象加入数组
        ...(showGpuAnalysis
          ? [
              {
                title: t('navigation.gpuAnalysis'),
                url: '/admin/gpu-analysis',
                icon: GpuIcon,
              },
            ]
          : []),
      ],
    },
    {
      title: t('sidebar.usersAndAccounts'),
      items: [
        {
          title: t('navigation.userManagement'),
          url: '/admin/users',
          icon: UserRoundIcon,
        },
        {
          title: t('navigation.accountManagement'),
          url: '/admin/accounts',
          icon: UsersRoundIcon,
        },
      ],
    },
    {
      title: t('sidebar.dataAndImages'),
      items: [
        {
          title: t('navigation.imageManagement'),
          icon: BoxIcon,
          items: [
            {
              title: t('navigation.imageCreation'),
              url: '/admin/env/registry',
            },
            {
              title: t('navigation.imageList'),
              url: '/admin/env/images',
            },
          ],
        },
        {
          title: t('navigation.dataManagement'),
          icon: DatabaseIcon,
          url: '/admin/data',
        },
        {
          title: t('navigation.fileManagement'),
          icon: FolderIcon,
          url: '/admin/files',
        },
      ],
    },
    {
      title: t('navigation.more'),
      items: [
        {
          title: t('navigation.platformSettings'),
          icon: SettingsIcon,
          url: '/admin/more',
        },
        {
          title: t('navigation.approvalOrder'),
          url: '/admin/more/orders',
          icon: ClipboardCheckIcon,
        },
        {
          title: t('navigation.aboutCrater'),
          url: '/admin/more/version',
          icon: SettingsIcon,
        },
      ],
    },
  ]
}

function RouteComponent() {
  const groups = useAdminSidebarGroups()
  const pathname = useLocation({
    select: (location) => location.pathname,
  })

  return (
    <AppLayout groups={groups} rawPath={pathname}>
      <Outlet />
    </AppLayout>
  )
}

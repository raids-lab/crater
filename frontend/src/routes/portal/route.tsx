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
import { Outlet, createFileRoute, redirect, useLocation } from '@tanstack/react-router'
import {
  BarChartBigIcon,
  BoxIcon,
  DatabaseIcon,
  FlaskConicalIcon,
  FolderIcon,
  SettingsIcon,
  ShoppingBagIcon,
  SquareChartGanttIcon,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import AppLayout from '@/components/layout/app-layout'
import NotFound from '@/components/placeholder/not-found'
import { NavGroupProps } from '@/components/sidebar/types'

export const Route = createFileRoute('/portal')({
  beforeLoad: ({ context, location }) => {
    if (!context.auth.isAuthenticated) {
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
  notFoundComponent: () => <NotFound />,
})

function RouteComponent() {
  const groups = useUserSidebarGroups()
  const pathname = useLocation({
    select: (location) => location.pathname,
  })

  return (
    <AppLayout groups={groups} rawPath={pathname}>
      <Outlet />
    </AppLayout>
  )
}

// 使用 hook 获取翻译版的侧边栏组
const useUserSidebarGroups = (): NavGroupProps[] => {
  const { t } = useTranslation()

  return [
    {
      title: t('sidebar.resourceAndMonitoring'),
      items: [
        {
          title: t('navigation.platformOverview'),
          url: 'overview',
          icon: SquareChartGanttIcon,
        },
        {
          title: t('navigation.clusterMonitoring'),
          icon: BarChartBigIcon,
          items: [
            {
              title: t('navigation.gpuMonitoring'),
              url: 'monitor/gpu',
            },
            {
              title: t('navigation.freeResources'),
              url: 'monitor/idle',
            },
            {
              title: t('navigation.networkMonitoring'),
              url: 'monitor/network',
            },
          ],
        },
      ],
    },
    {
      title: t('sidebar.jobsAndServices'),
      items: [
        {
          title: t('navigation.myJobs'),
          icon: FlaskConicalIcon,
          items: [
            {
              title: t('navigation.customJobs'),
              url: 'jobs/custom',
            },
            {
              title: t('navigation.jupyterLab'),
              url: 'jobs/inter',
            },
          ],
        },
        {
          title: t('navigation.jobTemplates'),
          url: 'templates',
          icon: ShoppingBagIcon,
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
              url: 'env/registry',
            },
            {
              title: t('navigation.imageList'),
              url: 'env/images',
            },
          ],
        },
        {
          title: t('navigation.dataManagement'),
          icon: DatabaseIcon,
          items: [
            {
              title: t('navigation.datasets'),
              url: 'data/datasets',
            },
            {
              title: t('navigation.models'),
              url: 'data/models',
            },
            {
              title: t('navigation.blocks'),
              url: 'data/blocks',
            },
          ],
        },
        {
          title: t('navigation.fileManagement'),
          icon: FolderIcon,
          url: 'files',
        },
      ],
    },
    {
      title: t('sidebar.others'),
      items: [
        {
          title: t('navigation.settings'),
          icon: SettingsIcon,
          items: [
            {
              title: t('navigation.userSettings'),
              url: 'settings/user',
            },
          ],
        },
      ],
    },
  ]
}

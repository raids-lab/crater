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

import { PathInfo } from '@/utils/title'

export const adminPath: PathInfo = {
  path: 'admin',
  titleKey: 'navigation.admin',
  isEmpty: true,
  children: [
    {
      path: 'cluster',
      titleKey: 'navigation.resourceManagement',
      isEmpty: true,
      children: [
        {
          path: 'node',
          titleKey: 'navigation.nodeManagement',
        },
        {
          path: 'label',
          titleKey: 'navigation.nodeLabels',
        },
        {
          path: 'resource',
          titleKey: 'navigation.resourceManagement',
        },
      ],
    },
    {
      path: 'monitor',
      titleKey: 'navigation.clusterMonitoring',
      isEmpty: true,
      children: [
        {
          path: 'network',
          titleKey: 'navigation.networkMonitoring',
        },
        {
          path: 'io',
          titleKey: 'navigation.ioMonitoring',
        },
      ],
    },
    {
      path: 'account',
      titleKey: 'navigation.accountManagement',
    },
    {
      path: 'user',
      titleKey: 'navigation.userManagement',
    },
    {
      path: 'job',
      titleKey: 'navigation.jobManagement',
    },
    {
      path: 'cron',
      titleKey: 'navigation.cronPolicy',
    },
    {
      path: 'image',
      titleKey: 'navigation.imageManagement',
      isEmpty: true,
      children: [
        {
          path: 'createimage',
          titleKey: 'navigation.imageCreation',
        },
        {
          path: 'uploadimage',
          titleKey: 'navigation.imageList',
        },
      ],
    },
    {
      path: 'data',
      titleKey: 'navigation.dataManagement',
      isEmpty: true,
      children: [
        {
          path: 'user',
          titleKey: 'navigation.userFiles',
        },
        {
          path: 'account',
          titleKey: 'navigation.accountFiles',
        },
        {
          path: 'dataset',
          titleKey: 'navigation.datasets',
        },
      ],
    },
    {
      path: 'setting',
      titleKey: 'navigation.platformSettings',
    },
  ],
}

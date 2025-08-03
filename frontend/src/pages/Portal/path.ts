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

export const craterPath: PathInfo = {
  path: 'portal',
  titleKey: 'navigation.portal',
  isEmpty: true,
  children: [
    {
      path: 'overview',
      titleKey: 'navigation.platformOverview',
      titleNavKey: 'navigation.platformFullName',
    },
    {
      path: 'monitor',
      titleKey: 'navigation.clusterMonitoring',
      isEmpty: true,
      children: [
        {
          path: 'gpu',
          titleKey: 'navigation.gpuMonitoring',
        },
        {
          path: 'node',
          titleKey: 'navigation.freeResources',
        },
        {
          path: 'network',
          titleKey: 'navigation.networkMonitoring',
        },
      ],
    },
    {
      path: 'job',
      titleKey: 'navigation.myJobs',
      isEmpty: true,
      children: [
        {
          path: 'batch',
          titleKey: 'navigation.customJobs',
          children: [
            {
              path: 'new-vcjobs',
              titleKey: 'navigation.createCustomJob',
            },
            {
              path: 'new-aijobs',
              titleKey: 'navigation.createCustomJob',
            },
            {
              path: 'new-spjobs',
              titleKey: 'navigation.createCustomJob',
            },
            {
              path: 'new-tensorflow',
              titleKey: 'navigation.createTensorflowJob',
            },
            {
              path: 'new-pytorch',
              titleKey: 'navigation.createPytorchJob',
            },
          ],
        },
        {
          path: 'inter',
          titleKey: 'navigation.jupyterLab',
          children: [
            {
              path: 'new-jupyter-vcjobs',
              titleKey: 'navigation.createJupyterLab',
            },
            {
              path: 'new-jupyter-aijobs',
              titleKey: 'navigation.createJupyterLab',
            },
          ],
        },
      ],
    },
    {
      path: 'modal',
      titleKey: 'navigation.jobTemplates',
      isEmpty: true,
    },
    {
      path: 'image',
      titleKey: 'navigation.myImages',
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
          path: 'filesystem',
          titleKey: 'navigation.fileSystem',
        },
        {
          path: 'dataset',
          titleKey: 'navigation.datasets',
        },
        {
          path: 'model',
          titleKey: 'navigation.models',
        },
        {
          path: 'sharefile',
          titleKey: 'navigation.blocks',
        },
      ],
    },
    {
      path: 'files',
      titleKey: 'navigation.fileManagement',
      isEmpty: true,
      children: [
        {
          path: 'spacefile',
          titleKey: 'navigation.spaceFile',
        },
      ],
    },
    {
      path: 'account',
      titleKey: 'navigation.accountManagement',
      isEmpty: true,
      children: [
        {
          path: 'member',
          titleKey: 'navigation.memberManagement',
        },
      ],
    },
    {
      path: 'setting',
      titleKey: 'navigation.settings',
      children: [
        {
          path: 'user',
          titleKey: 'navigation.userSettings',
        },
        {
          path: 'platform',
          titleKey: 'navigation.platformSettings',
        },
      ],
    },
  ],
}

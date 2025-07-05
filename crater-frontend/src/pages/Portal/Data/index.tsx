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

import { RouteObject } from 'react-router-dom'
import { apiGetDataset } from '@/services/api/dataset'
import { DataView } from '@/components/custom/DataView'

const datasetRoutes: RouteObject[] = [
  {
    index: true,
    element: <DataView apiGetDataset={apiGetDataset} sourceType="dataset" />,
  },
  {
    path: ':id',
    lazy: () => import('./DatasetShare'),
  },
]

const modelRoutes: RouteObject[] = [
  {
    index: true,
    element: <DataView apiGetDataset={apiGetDataset} sourceType="model" />,
  },
  {
    path: ':id',
    lazy: () => import('./ModelShare'),
  },
]

const shareFileRoutes: RouteObject[] = [
  {
    index: true,
    element: <DataView apiGetDataset={apiGetDataset} sourceType="sharefile" />, // 传入不同的resourceType
  },
  {
    path: ':id',
    lazy: () => import('./ShareFileShare'), // 确保返回符合LazyRouteFunction类型的组件加载器
  },
]

export { datasetRoutes, modelRoutes, shareFileRoutes }

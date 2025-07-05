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
import VolcanoOverview from './Volcano/Overview'
import { globalJobUrl, store } from '@/utils/store'
import ColocateOverview from './Colocate/Overview'
import { Base } from '../Detail/Base'

const jobType = store.get(globalJobUrl)
const batchRoutes: RouteObject[] = [
  {
    index: true,
    element: jobType === 'aijobs' ? <ColocateOverview /> : <VolcanoOverview />,
  },
  {
    path: ':name',
    element: <Base />,
  },
  {
    path: 'new-aijobs',
    lazy: () => import('../New/EmiasCustom'),
  },
  {
    path: 'new-vcjobs',
    lazy: () => import('../New/Custom'),
  },
  {
    path: 'new-tensorflow',
    lazy: () => import('../New/Tensorflow'),
  },
  {
    path: 'new-pytorch',
    lazy: () => import('../New/Pytorch'),
  },
  {
    path: 'new-spjobs',
    lazy: () => import('../New/Sparse'),
  },
]

export default batchRoutes

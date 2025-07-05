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
import InterOverview from './InterOverview'
import { Base } from '../Detail/Base'

const interactiveRoutes: RouteObject[] = [
  {
    index: true,
    element: <InterOverview />,
  },
  {
    path: 'new-jupyter-vcjobs',
    lazy: () => import('../New/Jupyter'),
  },
  {
    path: 'new-jupyter-aijobs',
    lazy: () => import('../New/EmiasJupyter'),
  },
  {
    path: ':name',
    element: <Base />,
  },
]

export default interactiveRoutes

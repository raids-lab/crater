/**
 * Copyright 2026 The Crater Project Team, RAIDS-Lab
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
import { createFileRoute } from '@tanstack/react-router'
import { t as i18nT } from 'i18next'

import TensorboardPanelList from '@/components/tensorboard/tensorboard-panel-list'

export const Route = createFileRoute('/portal/jobs/tensorboard/')({
  loader: () => ({
    crumb: i18nT('tensorboard.list.crumb'),
  }),
  component: TensorboardPanelList,
})

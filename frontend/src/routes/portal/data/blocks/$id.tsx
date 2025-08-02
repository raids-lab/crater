import { createFileRoute } from '@tanstack/react-router'

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
import { SharedResourceTable } from '@/components/file/data-detail'
import { detailValidateSearch } from '@/components/layout/detail-page'

import {
  apiCancelShareWithQueue,
  apiCancelShareWithUser,
  apiDatasetDelete,
  apiShareDatasetwithQueue,
  apiShareDatasetwithUser,
} from '@/services/api/dataset'

export const Route = createFileRoute('/portal/data/blocks/$id')({
  validateSearch: detailValidateSearch,
  component: RouteComponent,
})

function RouteComponent() {
  const { tab } = Route.useSearch()
  const navigate = Route.useNavigate()
  return (
    <SharedResourceTable
      resourceType="sharefile"
      id={Route.useParams().id}
      apiShareDatasetwithQueue={apiShareDatasetwithQueue}
      apiShareDatasetwithUser={apiShareDatasetwithUser}
      apiCancelDatasetSharewithQueue={apiCancelShareWithQueue}
      apiCancelDatasetSharewithUser={apiCancelShareWithUser}
      apiDatasetDelete={apiDatasetDelete}
      currentTab={tab}
      setCurrentTab={(tab) => navigate({ to: '.', search: { tab } })}
    />
  )
}

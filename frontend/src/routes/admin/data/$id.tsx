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
import { detailLinkOptions, detailValidateSearch } from '@/components/layout/detail-page'

import {
  apiAdminCancelShareWithQueue,
  apiAdminCancelShareWithUser,
  apiAdminShareDatasetwithQueue,
  apiAdminShareDatasetwithUser,
  apiDatasetDelete,
} from '@/services/api/dataset'
import { queryAdminDataByID } from '@/services/query/data'

export const Route = createFileRoute('/admin/data/$id')({
  validateSearch: detailValidateSearch,
  component: DatasetDetail,
  loader: async ({ params, context: { queryClient } }) => {
    const { id } = params
    const datasetId = Number(id) || 0
    const { data } = await queryClient.ensureQueryData(queryAdminDataByID(datasetId))
    const dataset = data.length > 0 ? data[0] : undefined
    return {
      crumb: dataset?.name || id,
      data: dataset,
    }
  },
})

function DatasetDetail() {
  const { tab } = Route.useSearch()
  const navigate = Route.useNavigate()
  return (
    <SharedResourceTable
      resourceType="dataset"
      data={Route.useLoaderData().data}
      apiShareDatasetwithQueue={apiAdminShareDatasetwithQueue}
      apiShareDatasetwithUser={apiAdminShareDatasetwithUser}
      apiCancelDatasetSharewithQueue={apiAdminCancelShareWithQueue}
      apiCancelDatasetSharewithUser={apiAdminCancelShareWithUser}
      apiDatasetDelete={apiDatasetDelete}
      currentTab={tab}
      setCurrentTab={(tab) => navigate(detailLinkOptions(tab))}
    />
  )
}

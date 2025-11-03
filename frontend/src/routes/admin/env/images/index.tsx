import { createFileRoute } from '@tanstack/react-router'
import { t } from 'i18next'

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
// i18n-processed-v1.1.0 (no translatable strings)
import { ImageListTable } from '@/components/image/images'

import {
  apiAdminChangeImageDescription,
  apiAdminChangeImagePublicStatus,
  apiAdminDeleteImageList,
  apiAdminListImage,
} from '@/services/api/admin/imagepack'

export const Route = createFileRoute('/admin/env/images/')({
  component: RouteComponent,
  loader: () => {
    return {
      crumb: t('navigation.imageList'),
    }
  },
})

function RouteComponent() {
  return (
    <ImageListTable
      apiListImage={apiAdminListImage}
      apiDeleteImageList={apiAdminDeleteImageList}
      apiChangeImagePublicStatus={apiAdminChangeImagePublicStatus}
      apiChangeImageDescription={apiAdminChangeImageDescription}
      isAdminMode={true}
    />
  )
}

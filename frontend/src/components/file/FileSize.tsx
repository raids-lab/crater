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

// i18n-processed-v1.1.0
// Modified code
import { useTranslation } from 'react-i18next'

export const FileSizeComponent = ({ size }: { size: number }) => {
  const { t } = useTranslation()

  const formatSize = (size: number) => {
    if (size < 1024) {
      return size + t('fileSize.bytes')
    } else if (size < 1024 * 1024) {
      return (size / 1024).toFixed(2) + t('fileSize.kilobytes')
    } else if (size < 1024 * 1024 * 1024) {
      return (size / (1024 * 1024)).toFixed(2) + t('fileSize.megabytes')
    } else {
      return (size / (1024 * 1024 * 1024)).toFixed(2) + t('fileSize.gigabytes')
    }
  }

  return <>{formatSize(size)}</>
}

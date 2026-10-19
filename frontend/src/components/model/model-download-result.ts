// Copyright 2026 The Crater Project Team, RAIDS-Lab
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
import type { TFunction } from 'i18next'

import type { CreateModelDownloadReq, ModelDownload } from '@/services/api/modeldownload'

type DownloadResult = Pick<ModelDownload, 'source' | 'revision' | 'status' | 'path'>

export function getModelDownloadReuseNotice(
  request: Pick<CreateModelDownloadReq, 'source' | 'revision'>,
  download: DownloadResult,
  t: TFunction
) {
  const requestedSource = request.source || 'modelscope'
  const requestedRevision = request.revision || ''
  const revisionChanged = requestedRevision !== download.revision
  if (requestedSource === download.source && !revisionChanged) return null

  const revisionLabel = (revision: string) =>
    revision || t('modelDownload.dialog.unspecifiedRevision')

  return {
    title: t(
      revisionChanged
        ? 'modelDownload.dialog.reuseRevisionWarning'
        : 'modelDownload.dialog.reuseSourceWarning'
    ),
    description: t('modelDownload.dialog.reuseDetails', {
      requestedSource,
      requestedRevision: revisionLabel(requestedRevision),
      actualSource: download.source,
      actualRevision: revisionLabel(download.revision),
      status: download.status,
      path: download.path,
    }),
  }
}

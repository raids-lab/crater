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
import { apiV1Post } from '@/services/client'

import { IResponse } from '../../types'

export interface IDlAnalyze {
  p100: {
    gpuUtilAvg: number
    gpuMemoryMaxGB: number
  }
  v100: {
    gpuUtilAvg: number
    gpuMemoryMaxGB: number
    smActiveAvg: number
    smOccupancyAvg: number
    fp32ActiveAvg: number
    dramActiveAvg: number
  }
}

interface IDlAnalyzeRequest {
  runningType: string
  relationShips: string[]
  macs: number
  params: number
  batchSize: number
  vocabularySize: number[]
  embeddingDim: number[]
}

// /v1/recommenddljob/analyze
export const apiDlAnalyze = (data: IDlAnalyzeRequest) =>
  apiV1Post<IResponse<IDlAnalyze>>('recommenddljob/analyze', {
    ...data,
  })

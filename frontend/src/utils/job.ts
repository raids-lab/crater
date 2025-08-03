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
import { JobType } from '@/services/api/vcjob'

/**
 * Get the URL for creating a new job based on an existing job type
 * @param jobType The type of job
 * @returns The URL path for creating a new job
 */
export const getNewJobUrl = (jobType: JobType): string => {
  switch (jobType) {
    case JobType.Jupyter:
      return '/portal/job/inter/new-jupyter-vcjobs'
    case JobType.Custom:
      return '/portal/job/batch/new-vcjobs'
    case JobType.Tensorflow:
      return '/portal/job/batch/new-tensorflow'
    case JobType.Pytorch:
      return '/portal/job/batch/new-pytorch'
    default:
      return '/portal/job/batch/new-vcjobs'
  }
}

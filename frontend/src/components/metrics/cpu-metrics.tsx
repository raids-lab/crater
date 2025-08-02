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
import { ProfileData } from '@/services/api/vcjob'

import { MetricCard } from '../ui-custom/metric-card'
import { MetricGroup } from '../ui-custom/metric-group'

interface CpuMetricsProps {
  profileData: ProfileData
}

export default function CpuMetrics({ profileData }: CpuMetricsProps) {
  const hasCpuUsageData =
    profileData.cpu_usage_avg !== undefined ||
    profileData.cpu_usage_max !== undefined ||
    profileData.cpu_usage_std !== undefined

  const hasCpuMemData =
    profileData.cpu_mem_avg !== undefined ||
    profileData.cpu_mem_max !== undefined ||
    profileData.cpu_mem_std !== undefined

  if (!hasCpuUsageData && !hasCpuMemData) return null

  return (
    <div className="space-y-4">
      {hasCpuUsageData && (
        <MetricGroup title="CPU Usage">
          {profileData.cpu_usage_avg !== undefined && (
            <MetricCard
              title="Average CPU Usage"
              value={profileData.cpu_usage_avg}
              unit="%"
              description="Average CPU utilization"
            />
          )}
          {profileData.cpu_usage_max !== undefined && (
            <MetricCard
              title="Maximum CPU Usage"
              value={profileData.cpu_usage_max}
              unit="%"
              description="Peak CPU utilization"
            />
          )}
          {profileData.cpu_usage_std !== undefined && (
            <MetricCard
              title="CPU Usage Std Dev"
              value={profileData.cpu_usage_std}
              unit=""
              description="Standard deviation of CPU usage"
            />
          )}
        </MetricGroup>
      )}

      {hasCpuMemData && (
        <MetricGroup title="CPU Memory">
          {profileData.cpu_mem_avg !== undefined && (
            <MetricCard
              title="Average CPU Memory"
              value={profileData.cpu_mem_avg / 1024}
              unit="GB"
              description="Average memory usage"
            />
          )}
          {profileData.cpu_mem_max !== undefined && (
            <MetricCard
              title="Maximum CPU Memory"
              value={profileData.cpu_mem_max / 1024}
              unit="GB"
              description="Peak memory usage"
            />
          )}
          {profileData.cpu_mem_std !== undefined && (
            <MetricCard
              title="CPU Memory Std Dev"
              value={profileData.cpu_mem_std / 1024}
              unit="GB"
              description="Standard deviation of memory usage"
            />
          )}
        </MetricGroup>
      )}
    </div>
  )
}

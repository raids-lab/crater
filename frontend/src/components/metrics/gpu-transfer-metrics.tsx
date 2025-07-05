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

interface GpuTransferMetricsProps {
  profileData: ProfileData
}

export default function GpuTransferMetrics({ profileData }: GpuTransferMetricsProps) {
  const hasPcieData =
    profileData.pcie_tx_avg !== undefined ||
    profileData.pcie_tx_max !== undefined ||
    profileData.pcie_rx_avg !== undefined ||
    profileData.pcie_rx_max !== undefined

  if (!hasPcieData) return null

  return (
    <div className="space-y-4">
      <MetricGroup title="PCIe Transfer">
        {profileData.pcie_tx_avg !== undefined && (
          <MetricCard
            title="Average PCIe TX"
            value={profileData.pcie_tx_avg}
            unit="MB/s"
            description="Average PCIe transmission rate"
          />
        )}
        {profileData.pcie_tx_max !== undefined && (
          <MetricCard
            title="Maximum PCIe TX"
            value={profileData.pcie_tx_max}
            unit="MB/s"
            description="Peak PCIe transmission rate"
          />
        )}
        {profileData.pcie_rx_avg !== undefined && (
          <MetricCard
            title="Average PCIe RX"
            value={profileData.pcie_rx_avg}
            unit="MB/s"
            description="Average PCIe receive rate"
          />
        )}
        {profileData.pcie_rx_max !== undefined && (
          <MetricCard
            title="Maximum PCIe RX"
            value={profileData.pcie_rx_max}
            unit="MB/s"
            description="Peak PCIe receive rate"
          />
        )}
      </MetricGroup>
    </div>
  )
}

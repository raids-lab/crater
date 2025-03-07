import { ProfileData } from "@/services/api/vcjob";
import { MetricCard } from "../ui/metric-card";
import { MetricGroup } from "../ui/metric-group";

interface GpuTransferMetricsProps {
  profileData: ProfileData;
}

export default function GpuTransferMetrics({
  profileData,
}: GpuTransferMetricsProps) {
  const hasPcieData =
    profileData.pcie_tx_avg !== undefined ||
    profileData.pcie_tx_max !== undefined ||
    profileData.pcie_rx_avg !== undefined ||
    profileData.pcie_rx_max !== undefined;

  if (!hasPcieData) return null;

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
  );
}

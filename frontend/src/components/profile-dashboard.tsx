import { MetricSection } from "./ui/metric-section";
import { ProgressCard } from "./ui/progress-card";
import { MetricCard } from "./ui/metric-card";
import { Cpu, MemoryStickIcon as Memory, ArrowDownUp } from "lucide-react";
import { ProfileData } from "@/services/api/vcjob";
import GpuIcon from "./icon/GpuIcon";

interface ProfileDashboardProps {
  profileData: ProfileData;
}

export default function ProfileDashboard({
  profileData,
}: ProfileDashboardProps) {
  // Check if CPU metrics exist
  const hasCpuMetrics =
    profileData.cpu_usage_avg !== undefined ||
    profileData.cpu_usage_max !== undefined ||
    profileData.cpu_usage_std !== undefined ||
    profileData.cpu_mem_avg !== undefined ||
    profileData.cpu_mem_max !== undefined ||
    profileData.cpu_mem_std !== undefined;

  // Check if GPU utilization metrics exist
  const hasGpuUtilMetrics =
    profileData.gpu_util_avg !== undefined ||
    profileData.gpu_util_max !== undefined ||
    profileData.gpu_util_std !== undefined;

  // Check if SM metrics exist
  const hasSmMetrics =
    profileData.sm_active_avg !== undefined ||
    profileData.sm_active_max !== undefined ||
    profileData.sm_active_std !== undefined ||
    profileData.sm_occupancy_avg !== undefined ||
    profileData.sm_occupancy_max !== undefined ||
    profileData.sm_occupancy_std !== undefined ||
    profileData.sm_util_std !== undefined;

  // Check if memory metrics exist
  const hasMemoryMetrics =
    profileData.dram_util_avg !== undefined ||
    profileData.dram_util_max !== undefined ||
    profileData.dram_util_std !== undefined ||
    profileData.mem_copy_util_avg !== undefined ||
    profileData.mem_copy_util_max !== undefined ||
    profileData.mem_copy_util_std !== undefined ||
    profileData.gpu_mem_max !== undefined;

  // Check if PCIe metrics exist
  const hasPcieMetrics =
    profileData.pcie_tx_avg !== undefined ||
    profileData.pcie_tx_max !== undefined ||
    profileData.pcie_rx_avg !== undefined ||
    profileData.pcie_rx_max !== undefined;

  return (
    <div className="space-y-8">
      {/* CPU Metrics Section */}
      {hasCpuMetrics && (
        <MetricSection title="CPU Metrics" icon={<Cpu className="h-5 w-5" />}>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
            {profileData.cpu_usage_avg !== undefined && (
              <MetricCard
                title="Average CPU Usage"
                value={profileData.cpu_usage_avg}
                unit="Core"
                description="Average CPU utilization"
              />
            )}
            {profileData.cpu_usage_max !== undefined && (
              <MetricCard
                title="Maximum CPU Usage"
                value={profileData.cpu_usage_max}
                unit="Core"
                description="Peak CPU utilization"
              />
            )}
            {profileData.cpu_usage_std !== undefined && (
              <MetricCard
                title="CPU Usage Std Dev"
                value={profileData.cpu_usage_std}
                unit="Core"
                description="Standard deviation of CPU usage"
              />
            )}
            {profileData.cpu_mem_avg !== undefined && (
              <MetricCard
                title="Average CPU Memory"
                value={profileData.cpu_mem_avg}
                unit="MB"
                description="Average memory usage"
              />
            )}
            {profileData.cpu_mem_max !== undefined && (
              <MetricCard
                title="Maximum CPU Memory"
                value={profileData.cpu_mem_max}
                unit="MB"
                description="Peak memory usage"
              />
            )}
            {profileData.cpu_mem_std !== undefined && (
              <MetricCard
                title="CPU Memory Std Dev"
                value={profileData.cpu_mem_std}
                unit="MB"
                description="Standard deviation of memory usage"
              />
            )}
          </div>
        </MetricSection>
      )}

      {/* GPU Utilization Section */}
      {hasGpuUtilMetrics && (
        <MetricSection
          title="GPU Utilization"
          icon={<GpuIcon className="h-5 w-5" />}
        >
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
            {profileData.gpu_util_avg !== undefined && (
              <ProgressCard
                title="Average GPU Utilization"
                value={profileData.gpu_util_avg}
                showPercentage={true}
                description="Average GPU utilization (0-1)"
              />
            )}
            {profileData.gpu_util_max !== undefined && (
              <ProgressCard
                title="Maximum GPU Utilization"
                value={profileData.gpu_util_max}
                showPercentage={true}
                description="Peak GPU utilization (0-1)"
              />
            )}
            {profileData.gpu_util_std !== undefined && (
              <MetricCard
                title="GPU Utilization Std Dev"
                value={profileData.gpu_util_std}
                unit=""
                description="Standard deviation of GPU utilization"
              />
            )}
          </div>
        </MetricSection>
      )}

      {/* SM Metrics Section */}
      {hasSmMetrics && (
        <MetricSection
          title="GPU Streaming Multiprocessor Metrics"
          icon={<GpuIcon className="h-5 w-5" />}
        >
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
            {profileData.sm_active_avg !== undefined && (
              <ProgressCard
                title="Average SM Active"
                value={profileData.sm_active_avg}
                showPercentage={true}
                description="Average SM activity (0-1)"
              />
            )}
            {profileData.sm_active_max !== undefined && (
              <ProgressCard
                title="Maximum SM Active"
                value={profileData.sm_active_max}
                showPercentage={true}
                description="Peak SM activity (0-1)"
              />
            )}
            {profileData.sm_active_std !== undefined && (
              <MetricCard
                title="SM Active Std Dev"
                value={profileData.sm_active_std}
                unit=""
                description="Standard deviation of SM activity"
              />
            )}
            {profileData.sm_occupancy_avg !== undefined && (
              <ProgressCard
                title="Average SM Occupancy"
                value={profileData.sm_occupancy_avg}
                showPercentage={true}
                description="Average SM occupancy (0-1)"
              />
            )}
            {profileData.sm_occupancy_max !== undefined && (
              <ProgressCard
                title="Maximum SM Occupancy"
                value={profileData.sm_occupancy_max}
                showPercentage={true}
                description="Peak SM occupancy (0-1)"
              />
            )}
            {profileData.sm_occupancy_std !== undefined && (
              <MetricCard
                title="SM Occupancy Std Dev"
                value={profileData.sm_occupancy_std}
                unit=""
                description="Standard deviation of SM occupancy"
              />
            )}
            {profileData.sm_util_std !== undefined && (
              <MetricCard
                title="SM Utilization Std Dev"
                value={profileData.sm_util_std}
                unit=""
                description="Standard deviation of SM utilization"
              />
            )}
          </div>
        </MetricSection>
      )}

      {/* Memory Utilization Section */}
      {hasMemoryMetrics && (
        <MetricSection
          title="GPU Memory Utilization"
          icon={<Memory className="h-5 w-5" />}
        >
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
            {profileData.dram_util_avg !== undefined && (
              <ProgressCard
                title="Average DRAM Utilization"
                value={profileData.dram_util_avg}
                showPercentage={true}
                description="Average DRAM utilization (0-1)"
              />
            )}
            {profileData.dram_util_max !== undefined && (
              <ProgressCard
                title="Maximum DRAM Utilization"
                value={profileData.dram_util_max}
                showPercentage={true}
                description="Peak DRAM utilization (0-1)"
              />
            )}
            {profileData.dram_util_std !== undefined && (
              <MetricCard
                title="DRAM Utilization Std Dev"
                value={profileData.dram_util_std}
                unit=""
                description="Standard deviation of DRAM utilization"
              />
            )}
            {profileData.mem_copy_util_avg !== undefined && (
              <ProgressCard
                title="Average Memory Copy Utilization"
                value={profileData.mem_copy_util_avg}
                showPercentage={true}
                description="Average memory copy utilization (0-1)"
              />
            )}
            {profileData.mem_copy_util_max !== undefined && (
              <ProgressCard
                title="Maximum Memory Copy Utilization"
                value={profileData.mem_copy_util_max}
                showPercentage={true}
                description="Peak memory copy utilization (0-1)"
              />
            )}
            {profileData.mem_copy_util_std !== undefined && (
              <MetricCard
                title="Memory Copy Utilization Std Dev"
                value={profileData.mem_copy_util_std}
                unit=""
                description="Standard deviation of memory copy utilization"
              />
            )}
            {profileData.gpu_mem_max !== undefined && (
              <MetricCard
                title="Maximum GPU Memory"
                value={profileData.gpu_mem_max}
                unit="MB"
                description="Peak GPU memory usage"
              />
            )}
          </div>
        </MetricSection>
      )}

      {/* PCIe Transfer Section */}
      {hasPcieMetrics && (
        <MetricSection
          title="PCIe Transfer"
          icon={<ArrowDownUp className="h-5 w-5" />}
        >
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
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
          </div>
        </MetricSection>
      )}
    </div>
  );
}

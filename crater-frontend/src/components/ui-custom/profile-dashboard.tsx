import { MetricSection } from "./metric-section";
import { ProgressCard } from "./progress-card";
import { MetricCard } from "./metric-card";
import { Cpu, MemoryStickIcon as Memory, ArrowDownUp } from "lucide-react";
import { ProfileData } from "@/services/api/vcjob";
import GpuIcon from "../icon/GpuIcon";

interface ProfileDashboardProps {
  profileData: ProfileData;
}

export const containsBaseMetrics = (profileData: ProfileData) => {
  return (
    profileData.cpu_usage_avg !== undefined ||
    profileData.cpu_usage_max !== undefined ||
    profileData.cpu_usage_std !== undefined ||
    profileData.cpu_mem_avg !== undefined ||
    profileData.cpu_mem_max !== undefined ||
    profileData.cpu_mem_std !== undefined
  );
};

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

  // Check if memory metrics exist
  const hasMemoryMetrics =
    profileData.dram_util_avg !== undefined ||
    profileData.dram_util_max !== undefined ||
    profileData.dram_util_std !== undefined ||
    profileData.mem_copy_util_avg !== undefined ||
    profileData.mem_copy_util_max !== undefined ||
    profileData.mem_copy_util_std !== undefined ||
    profileData.gpu_mem_max !== undefined;

  // Check if SM metrics exist
  const hasSmMetrics =
    profileData.sm_active_avg !== undefined ||
    profileData.sm_active_max !== undefined ||
    profileData.sm_active_std !== undefined ||
    profileData.sm_occupancy_avg !== undefined ||
    profileData.sm_occupancy_max !== undefined ||
    profileData.sm_occupancy_std !== undefined ||
    profileData.sm_util_std !== undefined;

  // Check if tensor metrics exist
  const hasTensorMetrics =
    profileData.tensor_active_avg !== undefined ||
    profileData.tensor_active_max !== undefined ||
    profileData.tensor_active_std !== undefined;

  // Check if fp metrics exist
  const hasFpMetrics =
    profileData.fp64_active_avg !== undefined ||
    profileData.fp64_active_max !== undefined ||
    profileData.fp64_active_std !== undefined ||
    profileData.fp32_active_avg !== undefined ||
    profileData.fp32_active_max !== undefined ||
    profileData.fp32_active_std !== undefined ||
    profileData.fp16_active_avg !== undefined ||
    profileData.fp16_active_max !== undefined ||
    profileData.fp16_active_std !== undefined;

  // Check if PCIe metrics exist
  const hasPcieMetrics =
    profileData.pcie_tx_avg !== undefined ||
    profileData.pcie_tx_max !== undefined ||
    profileData.pcie_rx_avg !== undefined ||
    profileData.pcie_rx_max !== undefined;

  return (
    <div className="space-y-6">
      {/* CPU Metrics Section */}
      {hasCpuMetrics && (
        <MetricSection title="CPU 相关指标" icon={<Cpu className="h-5 w-5" />}>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
            {profileData.cpu_usage_avg !== undefined && (
              <ProgressCard
                title="Average CPU Usage"
                value={profileData.cpu_usage_avg}
                unit="Core"
                max={profileData.cpu_limit}
                showPercentage={false}
                description="作业运行期间 CPU 平均使用量"
              />
            )}
            {profileData.cpu_usage_max !== undefined && (
              <ProgressCard
                title="Maximum CPU Usage"
                value={profileData.cpu_usage_max}
                unit="Core"
                max={profileData.cpu_limit}
                showPercentage={false}
                description="作业运行期间 CPU 峰值使用量"
              />
            )}
            {profileData.cpu_usage_std !== undefined && (
              <ProgressCard
                title="CPU Usage Std Dev"
                value={profileData.cpu_usage_std}
                unit="Core"
                max={profileData.cpu_limit}
                showPercentage={false}
                description="作业运行期间 CPU 使用量标准差"
              />
            )}
            {profileData.cpu_mem_avg !== undefined && (
              <ProgressCard
                title="Average CPU Memory"
                value={profileData.cpu_mem_avg}
                unit="MB"
                max={profileData.mem_limit}
                showPercentage={false}
                description="作业运行期间内存平均使用量"
              />
            )}
            {profileData.cpu_mem_max !== undefined && (
              <ProgressCard
                title="Maximum CPU Memory"
                value={profileData.cpu_mem_max}
                unit="MB"
                max={profileData.mem_limit}
                showPercentage={false}
                description="作业运行期间内存峰值使用量"
              />
            )}
            {profileData.cpu_mem_std !== undefined && (
              <ProgressCard
                title="CPU Memory Std Dev"
                value={profileData.cpu_mem_std}
                unit="MB"
                max={profileData.mem_limit}
                showPercentage={false}
                description="作业运行期间内存使用量标准差"
              />
            )}
          </div>
        </MetricSection>
      )}

      {/* GPU Utilization Section */}
      {hasGpuUtilMetrics && (
        <MetricSection
          title="GPU 利用率相关指标"
          icon={<GpuIcon className="h-5 w-5" />}
        >
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
            {profileData.gpu_util_avg !== undefined && (
              <ProgressCard
                title="Average GPU Utilization"
                value={profileData.gpu_util_avg}
                showPercentage={true}
                description="作业运行期间 GPU 平均利用率"
              />
            )}
            {profileData.gpu_util_max !== undefined && (
              <ProgressCard
                title="Maximum GPU Utilization"
                value={profileData.gpu_util_max}
                showPercentage={true}
                description="作业运行期间 GPU 峰值利用率"
              />
            )}
            {profileData.gpu_util_std !== undefined && (
              <ProgressCard
                title="GPU Utilization Std Dev"
                value={profileData.gpu_util_std}
                showPercentage={true}
                unit=""
                description="运行期间 GPU 利用率的标准差"
              />
            )}
          </div>
        </MetricSection>
      )}

      {/* Memory Utilization Section */}
      {hasMemoryMetrics && (
        <MetricSection
          title="GPU 内存相关指标"
          icon={<Memory className="h-5 w-5" />}
        >
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
            {profileData.gpu_mem_avg !== undefined && (
              <ProgressCard
                title="Average GPU Memory"
                value={profileData.gpu_mem_avg}
                unit="MB"
                showPercentage={false}
                max={profileData.gpu_mem_total}
                description="作业运行期间 GPU 内存平均使用量"
              />
            )}
            {profileData.gpu_mem_max !== undefined && (
              <ProgressCard
                title="Maximum GPU Memory"
                value={profileData.gpu_mem_max}
                unit="MB"
                showPercentage={false}
                max={profileData.gpu_mem_total}
                description="作业运行期间 GPU 内存峰值使用量"
              />
            )}
            {profileData.gpu_mem_std !== undefined && (
              <ProgressCard
                title="GPU Memory Std Dev"
                value={profileData.gpu_mem_std}
                unit="MB"
                showPercentage={false}
                max={profileData.gpu_mem_total}
                description="作业运行期间 GPU 内存使用量标准差"
              />
            )}
            {profileData.dram_util_avg !== undefined && (
              <ProgressCard
                title="Average DRAM Utilization"
                value={profileData.dram_util_avg}
                showPercentage={true}
                description="作业运行期间 DRAM 平均利用率"
              />
            )}
            {profileData.dram_util_max !== undefined && (
              <ProgressCard
                title="Maximum DRAM Utilization"
                value={profileData.dram_util_max}
                showPercentage={true}
                description="作业运行期间 DRAM 峰值利用率"
              />
            )}
            {profileData.dram_util_std !== undefined && (
              <ProgressCard
                title="DRAM Utilization Std Dev"
                value={profileData.dram_util_std}
                showPercentage={true}
                description="作业运行期间 DRAM 利用率标准差"
              />
            )}
            {profileData.mem_copy_util_avg !== undefined && (
              <ProgressCard
                title="Average Memory Copy Utilization"
                value={profileData.mem_copy_util_avg}
                showPercentage={true}
                description="作业运行期间内存拷贝平均利用率"
              />
            )}
            {profileData.mem_copy_util_max !== undefined && (
              <ProgressCard
                title="Maximum Memory Copy Utilization"
                value={profileData.mem_copy_util_max}
                showPercentage={true}
                description="作业运行期间内存拷贝峰值利用率"
              />
            )}
            {profileData.mem_copy_util_std !== undefined && (
              <ProgressCard
                title="Memory Copy Utilization Std Dev"
                value={profileData.mem_copy_util_std}
                showPercentage={true}
                description="作业运行期间内存拷贝利用率标准差"
              />
            )}
          </div>
        </MetricSection>
      )}

      {/* SM Metrics Section */}
      {hasSmMetrics && (
        <MetricSection
          title="GPU 流式多处理器相关指标"
          icon={<GpuIcon className="h-5 w-5" />}
        >
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
            {profileData.sm_active_avg !== undefined && (
              <ProgressCard
                title="Average SM Active"
                value={profileData.sm_active_avg}
                showPercentage={true}
                description="作业运行期间 GPU 流式多处理器平均活跃度"
              />
            )}
            {profileData.sm_active_max !== undefined && (
              <ProgressCard
                title="Maximum SM Active"
                value={profileData.sm_active_max}
                showPercentage={true}
                description="作业运行期间 GPU 流式多处理器峰值活跃度"
              />
            )}
            {profileData.sm_active_std !== undefined && (
              <ProgressCard
                title="SM Active Std Dev"
                value={profileData.sm_active_std}
                showPercentage={true}
                description="作业运行期间 GPU 流式多处理器活跃度标准差"
              />
            )}
            {profileData.sm_occupancy_avg !== undefined && (
              <ProgressCard
                title="Average SM Occupancy"
                value={profileData.sm_occupancy_avg}
                showPercentage={true}
                description="作业运行期间 GPU 流式多处理器平均占用率"
              />
            )}
            {profileData.sm_occupancy_max !== undefined && (
              <ProgressCard
                title="Maximum SM Occupancy"
                value={profileData.sm_occupancy_max}
                showPercentage={true}
                description="作业运行期间 GPU 流式多处理器峰值占用率"
              />
            )}
            {profileData.sm_occupancy_std !== undefined && (
              <ProgressCard
                title="SM Occupancy Std Dev"
                value={profileData.sm_occupancy_std}
                showPercentage={true}
                description="作业运行期间 GPU 流式多处理器占用率标准差"
              />
            )}
            {profileData.sm_util_avg !== undefined && (
              <MetricCard
                title="SM Utilization Std Dev"
                value={profileData.sm_util_avg}
                unit=""
                description="作业运行期间 GPU 流式多处理器平均利用率"
              />
            )}
            {profileData.sm_util_max !== undefined && (
              <MetricCard
                title="SM Utilization Std Dev"
                value={profileData.sm_util_max}
                unit=""
                description="作业运行期间 GPU 流式多处理器峰值利用率"
              />
            )}
            {profileData.sm_util_std !== undefined && (
              <MetricCard
                title="SM Utilization Std Dev"
                value={profileData.sm_util_std}
                unit=""
                description="作业运行期间 GPU 流式多处理器利用率标准差"
              />
            )}
          </div>
        </MetricSection>
      )}

      {/* Tensor Section */}
      {hasTensorMetrics && (
        <MetricSection
          title="Tensor 相关指标"
          icon={<GpuIcon className="h-5 w-5" />}
        >
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
            {profileData.tensor_active_avg !== undefined && (
              <ProgressCard
                title="Average Tensor Active"
                value={profileData.tensor_active_avg}
                showPercentage={true}
                description="作业运行期间 Tensor 平均活跃度"
              />
            )}
            {profileData.tensor_active_max !== undefined && (
              <ProgressCard
                title="Maximum Tensor Active"
                value={profileData.tensor_active_max}
                showPercentage={true}
                description="作业运行期间 Tensor 峰值活跃度"
              />
            )}
            {profileData.tensor_active_std !== undefined && (
              <ProgressCard
                title="Tensor Active Std Dev"
                value={profileData.tensor_active_std}
                showPercentage={true}
                description="作业运行期间 Tensor 活跃度标准差"
              />
            )}
          </div>
        </MetricSection>
      )}

      {/* FP Section */}
      {hasFpMetrics && (
        <MetricSection
          title="FP 相关指标"
          icon={<GpuIcon className="h-5 w-5" />}
        >
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
            {profileData.fp64_active_avg !== undefined && (
              <ProgressCard
                title="Average FP64 Active"
                value={profileData.fp64_active_avg}
                showPercentage={true}
                description="作业运行期间 FP64 平均活跃度"
              />
            )}
            {profileData.fp64_active_max !== undefined && (
              <ProgressCard
                title="Maximum FP64 Active"
                value={profileData.fp64_active_max}
                showPercentage={true}
                description="作业运行期间 FP64 峰值活跃度"
              />
            )}
            {profileData.fp64_active_std !== undefined && (
              <ProgressCard
                title="FP64 Active Std Dev"
                value={profileData.fp64_active_std}
                showPercentage={true}
                description="作业运行期间 FP64 活跃度标准差"
              />
            )}
            {profileData.fp32_active_avg !== undefined && (
              <ProgressCard
                title="Average FP32 Active"
                value={profileData.fp32_active_avg}
                showPercentage={true}
                description="作业运行期间 FP32 平均活跃度"
              />
            )}
            {profileData.fp32_active_max !== undefined && (
              <ProgressCard
                title="Maximum FP32 Active"
                value={profileData.fp32_active_max}
                showPercentage={true}
                description="作业运行期间 FP32 峰值活跃度"
              />
            )}
            {profileData.fp32_active_std !== undefined && (
              <ProgressCard
                title="FP32 Active Std Dev"
                value={profileData.fp32_active_std}
                showPercentage={true}
                description="作业运行期间 FP32 活跃度标准差"
              />
            )}
            {profileData.fp16_active_avg !== undefined && (
              <ProgressCard
                title="Average FP16 Active"
                value={profileData.fp16_active_avg}
                showPercentage={true}
                description="作业运行期间 FP16 平均活跃度"
              />
            )}
            {profileData.fp16_active_max !== undefined && (
              <ProgressCard
                title="Maximum FP16 Active"
                value={profileData.fp16_active_max}
                showPercentage={true}
                description="作业运行期间 FP16 峰值活跃度"
              />
            )}
            {profileData.fp16_active_std !== undefined && (
              <ProgressCard
                title="FP16 Active Std Dev"
                value={profileData.fp16_active_std}
                showPercentage={true}
                description="作业运行期间 FP16 活跃度标准差"
              />
            )}
          </div>
        </MetricSection>
      )}

      {/* PCIe Transfer Section */}
      {hasPcieMetrics && (
        <MetricSection
          title="PCIe 传输相关指标"
          icon={<ArrowDownUp className="h-5 w-5" />}
        >
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
            {profileData.pcie_tx_avg !== undefined && (
              <MetricCard
                title="Average PCIe TX"
                value={profileData.pcie_tx_avg}
                unit="MB/s"
                description="作业运行期间 PCIe 平均传输速率"
              />
            )}
            {profileData.pcie_tx_max !== undefined && (
              <MetricCard
                title="Maximum PCIe TX"
                value={profileData.pcie_tx_max}
                unit="MB/s"
                description="作业运行期间 PCIe 峰值传输速率"
              />
            )}
            {profileData.pcie_rx_avg !== undefined && (
              <MetricCard
                title="Average PCIe RX"
                value={profileData.pcie_rx_avg}
                unit="MB/s"
                description="作业运行期间 PCIe 平均接收速率"
              />
            )}
            {profileData.pcie_rx_max !== undefined && (
              <MetricCard
                title="Maximum PCIe RX"
                value={profileData.pcie_rx_max}
                unit="MB/s"
                description="作业运行期间 PCIe 峰值接收速率"
              />
            )}
          </div>
        </MetricSection>
      )}
    </div>
  );
}

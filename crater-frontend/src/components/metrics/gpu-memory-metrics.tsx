import { ProfileData } from "@/services/api/vcjob";
import { ProgressCard } from "../ui-custom/progress-card";
import { MetricGroup } from "../ui-custom/metric-group";
import { MetricCard } from "../ui-custom/metric-card";

interface GpuMemoryMetricsProps {
  profileData: ProfileData;
}

export default function GpuMemoryMetrics({
  profileData,
}: GpuMemoryMetricsProps) {
  const hasDramUtilData =
    profileData.dram_util_avg !== undefined ||
    profileData.dram_util_max !== undefined ||
    profileData.dram_util_std !== undefined;

  const hasMemCopyUtilData =
    profileData.mem_copy_util_avg !== undefined ||
    profileData.mem_copy_util_max !== undefined ||
    profileData.mem_copy_util_std !== undefined;

  if (
    !hasDramUtilData &&
    !hasMemCopyUtilData &&
    profileData.gpu_mem_max === undefined
  )
    return null;

  return (
    <div className="space-y-4">
      {hasDramUtilData && (
        <MetricGroup title="DRAM Utilization">
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
        </MetricGroup>
      )}

      {hasMemCopyUtilData && (
        <MetricGroup title="Memory Copy Utilization">
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
        </MetricGroup>
      )}

      {profileData.gpu_mem_max !== undefined && (
        <MetricGroup title="GPU Memory">
          <MetricCard
            title="Maximum GPU Memory"
            value={profileData.gpu_mem_max}
            unit="MB"
            description="Peak GPU memory usage"
          />
        </MetricGroup>
      )}
    </div>
  );
}

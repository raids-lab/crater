import { ProfileData } from "@/services/api/vcjob";
import { ProgressCard } from "../ui-custom/progress-card";
import { MetricGroup } from "../ui-custom/metric-group";
import { MetricCard } from "../ui-custom/metric-card";

interface GpuUtilizationMetricsProps {
  profileData: ProfileData;
}

export default function GpuUtilizationMetrics({
  profileData,
}: GpuUtilizationMetricsProps) {
  const hasGpuUtilData =
    profileData.gpu_util_avg !== undefined ||
    profileData.gpu_util_max !== undefined ||
    profileData.gpu_util_std !== undefined;

  const hasSmActiveData =
    profileData.sm_active_avg !== undefined ||
    profileData.sm_active_max !== undefined ||
    profileData.sm_active_std !== undefined;

  const hasSmOccupancyData =
    profileData.sm_occupancy_avg !== undefined ||
    profileData.sm_occupancy_max !== undefined ||
    profileData.sm_occupancy_std !== undefined;

  if (
    !hasGpuUtilData &&
    !hasSmActiveData &&
    !hasSmOccupancyData &&
    !profileData.sm_util_std
  )
    return null;

  return (
    <div className="space-y-4">
      {hasGpuUtilData && (
        <MetricGroup title="GPU Utilization">
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
        </MetricGroup>
      )}

      {hasSmActiveData && (
        <MetricGroup title="SM Active">
          {profileData.sm_active_avg !== undefined && (
            <ProgressCard
              title="Average SM Active"
              value={profileData.sm_active_avg}
              showPercentage={true}
              description="Average Streaming Multiprocessor activity (0-1)"
            />
          )}
          {profileData.sm_active_max !== undefined && (
            <ProgressCard
              title="Maximum SM Active"
              value={profileData.sm_active_max}
              showPercentage={true}
              description="Peak Streaming Multiprocessor activity (0-1)"
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
        </MetricGroup>
      )}

      {hasSmOccupancyData && (
        <MetricGroup title="SM Occupancy">
          {profileData.sm_occupancy_avg !== undefined && (
            <ProgressCard
              title="Average SM Occupancy"
              value={profileData.sm_occupancy_avg}
              showPercentage={true}
              description="Average Streaming Multiprocessor occupancy (0-1)"
            />
          )}
          {profileData.sm_occupancy_max !== undefined && (
            <ProgressCard
              title="Maximum SM Occupancy"
              value={profileData.sm_occupancy_max}
              showPercentage={true}
              description="Peak Streaming Multiprocessor occupancy (0-1)"
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
        </MetricGroup>
      )}

      {profileData.sm_util_std !== undefined && (
        <MetricGroup title="SM Utilization">
          <MetricCard
            title="SM Utilization Std Dev"
            value={profileData.sm_util_std}
            unit=""
            description="Standard deviation of SM utilization"
          />
        </MetricGroup>
      )}
    </div>
  );
}

import { LockIcon } from "lucide-react";
import TooltipLink from "@/components/label/TooltipLink";
import { getJobStateType, IJobInfo, JobStatus } from "@/services/api/vcjob";
import TipBadge from "../badge/TipBadge";

interface JobNameCellProps {
  jobInfo: IJobInfo;
}

export const JobNameCell = ({ jobInfo }: JobNameCellProps) => {
  return (
    <TooltipLink
      name={
        <div className="flex flex-row items-center">
          <p className="max-w-36 truncate">{jobInfo.name}</p>
          {jobInfo.keepWhenLowUsage && (
            <LockIcon className="text-muted-foreground ml-1 size-4" />
          )}
        </div>
      }
      to={
        getJobStateType(jobInfo.status) === JobStatus.NotStarted
          ? `${jobInfo.jobName}?tab=event`
          : jobInfo.jobName
      }
      tooltip={
        // `查看 ${jobInfo.name} 详情` +
        // (jobInfo.keepWhenLowUsage ? "（已锁定，低利用率仍保留）" : "")
        <div className="flex flex-row items-center justify-between gap-1.5">
          <p className="text-xs">查看 {jobInfo.name} 详情</p>
          {jobInfo.keepWhenLowUsage && (
            <TipBadge
              title="作业清理豁免中"
              className="text-primary bg-primary-foreground z-10"
            />
          )}
        </div>
      }
    />
  );
};

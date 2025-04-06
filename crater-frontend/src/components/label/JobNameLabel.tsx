import { useState } from "react";
import { LockIcon } from "lucide-react";
import TooltipLink from "@/components/label/TooltipLink";
import { getJobStateType, IJobInfo, JobStatus } from "@/services/api/vcjob";
import TipBadge from "../badge/TipBadge";
import { TimeDistance } from "../custom/TimeDistance";

interface JobNameCellProps {
  jobInfo: IJobInfo;
}

export const JobNameCell = ({ jobInfo }: JobNameCellProps) => {
  const [showLockTip, setShowLockTip] = useState(false);

  return (
    // 使用 relative 作为定位容器，确保 tooltip 的绝对定位生效
    <div className="relative flex items-center">
      <TooltipLink
        name={
          <div className="flex flex-row items-center">
            <p className="max-w-36 truncate">{jobInfo.name}</p>
            {jobInfo.locked && (
              <LockIcon
                className="text-muted-foreground ml-1 h-4 w-4"
                onMouseEnter={() => setShowLockTip(true)}
                onMouseLeave={() => setShowLockTip(false)}
              />
            )}
          </div>
        }
        to={
          getJobStateType(jobInfo.status) === JobStatus.NotStarted
            ? `${jobInfo.jobName}?tab=event`
            : jobInfo.jobName
        }
        tooltip={
          <div className="flex flex-row items-center justify-between gap-1.5">
            <p className="text-xs">查看 {jobInfo.jobName} 详情</p>
            {jobInfo.locked && (
              <TipBadge
                title="作业清理豁免中"
                className="text-primary bg-primary-foreground z-10"
              />
            )}
          </div>
        }
      />
      {jobInfo.locked && showLockTip && (
        <div className="bg-primary-foreground text-primary absolute top-full left-1/2 z-10 mb-1 -translate-x-1/2 transform rounded-lg px-3 py-1 text-xs whitespace-nowrap shadow-md">
          {jobInfo.permanentLocked ? (
            "永久锁定"
          ) : (
            <>
              锁定至 <TimeDistance date={jobInfo.lockedTimestamp} />
            </>
          )}
        </div>
      )}
    </div>
  );
};

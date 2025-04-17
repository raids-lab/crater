import { MetricSection } from "../ui-custom/metric-section";
import { MetricCard } from "../ui-custom/metric-card";
import { ClockIcon } from "lucide-react";
import { IJupyterDetail } from "@/services/api/vcjob";
import { Fragment, useMemo } from "react";
import {
  formatDistanceStrict,
  formatDuration,
  intervalToDuration,
} from "date-fns";
import { zhCN } from "date-fns/locale";
import { TerminatedSection } from "./TerminatedSection";

// 辅助函数：将 "32.671s" 或 "2m32.671s" 转换为秒数
const parseDurationString = (durationStr: string): number => {
  if (!durationStr) return 0;

  let totalSeconds = 0;

  // 处理分钟部分
  const minutesMatch = durationStr.match(/(\d+)m/);
  if (minutesMatch) {
    totalSeconds += parseInt(minutesMatch[1], 10) * 60;
  }

  // 处理秒部分
  const secondsMatch = durationStr.match(/(\d+\.?\d*)s/);
  if (secondsMatch) {
    totalSeconds += parseFloat(secondsMatch[1]);
  }

  return totalSeconds;
};

// 辅助函数：将秒数转换为易读的持续时间字符串
const formatSecondsToDuration = (seconds: number): string => {
  const duration = intervalToDuration({ start: 0, end: seconds * 1000 });

  return formatDuration(duration, {
    locale: zhCN,
    format: ["hours", "minutes", "seconds"],
    zero: false,
    delimiter: " ",
  });
};

// 新增组件：作业调度信息区域
interface ScheduleInfoSectionProps {
  data: IJupyterDetail;
}

export const JobInfoSections = ({
  data: { terminatedStates, ...props },
}: ScheduleInfoSectionProps) => {
  const scheduleInfo = useMemo(() => {
    const createdAt = props.createdAt ? new Date(props.createdAt) : null;
    const startedAt = props.startedAt ? new Date(props.startedAt) : null;
    const finishedAt = props.completedAt ? new Date(props.completedAt) : null;
    const imagePullTime = props.scheduleData?.imagePullTime || "";

    const updateTimeDiff = (later: Date, earlier: Date) => {
      const timeDifference = formatDistanceStrict(later, earlier, {
        locale: zhCN,
        addSuffix: false,
      });
      return timeDifference;
    };

    if (createdAt && startedAt && finishedAt) {
      // 运行时间（开始到结束）
      const runTime = updateTimeDiff(finishedAt, startedAt);

      // 总等待时间（创建到开始）
      const totalWaitTime = updateTimeDiff(startedAt, createdAt);

      // 镜像拉取时间（从字符串解析）
      const imagePullSeconds = parseDurationString(imagePullTime);
      const pullTime = formatSecondsToDuration(imagePullSeconds);

      // 排队时间（总等待时间 - 镜像拉取时间）
      const totalWaitSeconds =
        (startedAt.getTime() - createdAt.getTime()) / 1000;
      const queueSeconds = Math.max(0, totalWaitSeconds - imagePullSeconds);
      const queueTime = formatSecondsToDuration(queueSeconds);

      return {
        runTime,
        pullTime,
        waitTime: queueTime,
        totalWaitTime, // 如果需要也可以返回总等待时间
      };
    }
    return {
      runTime: "",
      pullTime: "",
      waitTime: "",
    };
  }, [props.completedAt, props.createdAt, props.startedAt, props.scheduleData]);

  return (
    <>
      {terminatedStates &&
        terminatedStates.length > 0 &&
        terminatedStates.map((state, index) => (
          <Fragment key={index}>
            <TerminatedSection state={state} index={index} />
          </Fragment>
        ))}
      <MetricSection
        title="作业调度信息"
        icon={<ClockIcon className="h-5 w-5" />}
      >
        <MetricCard
          title="作业排队用时"
          value={scheduleInfo.waitTime}
          unit=""
          description="作业从创建到被调度到节点的时间"
        />
        {!!scheduleInfo.pullTime && (
          <MetricCard
            title="镜像拉取用时"
            value={scheduleInfo.pullTime}
            unit=""
            description="镜像拉取所花费的时间，为空说明镜像已缓存"
          />
        )}
        <MetricCard
          title="作业运行用时"
          value={scheduleInfo.runTime}
          unit=""
          description="作业执行的总时长"
        />
      </MetricSection>
    </>
  );
};

import { ImagePackStatus } from "@/services/api/imagepack";
import { PhaseBadge, PhaseBadgeData } from "./PhaseBadge";

export const getImagePackStatusLabel = (
  status: ImagePackStatus,
): PhaseBadgeData => {
  switch (status) {
    case "Initial":
      return {
        label: "检查中",
        color: "text-highlight-purple bg-highlight-purple/20",
        description: "镜像包正在初始化检查",
      };
    case "Pending":
      return {
        label: "等待中",
        color: "text-highlight-slate bg-highlight-slate/20",
        description: "镜像包等待处理",
      };
    case "Running":
      return {
        label: "运行中",
        color: "text-highlight-sky bg-highlight-sky/20",
        description: "镜像包正在处理",
      };
    case "Finished":
      return {
        label: "成功",
        color: "text-highlight-emerald bg-highlight-emerald/20",
        description: "镜像包处理完成",
      };
    case "Failed":
      return {
        label: "失败",
        color: "text-highlight-red bg-highlight-red/20",
        description: "镜像包处理失败",
      };
    default:
      return {
        label: "未知",
        color: "text-highlight-slate bg-highlight-slate/20",
        description: "未知状态",
      };
  }
};

const ImageStatusBadge = ({ status }: { status: ImagePackStatus }) => {
  return <PhaseBadge phase={status} getPhaseLabel={getImagePackStatusLabel} />;
};

export default ImageStatusBadge;

import { PhaseBadge, PhaseBadgeData } from "./PhaseBadge";

export enum Visibility {
  Public = "public",
  Private = "private",
}

export const visibilityTypes = [
  {
    value: Visibility.Public,
    label: "公共",
    color: "text-highlight-green bg-highlight-green/20",
    description: "所有用户均可查看和使用的资源",
  },
  {
    value: Visibility.Private,
    label: "私有",
    color: "text-highlight-sky bg-highlight-sky/20",
    description: "仅限创建者及被授权用户可查看和使用的资源",
  },
];

const getVisibilityLabel = (visibility: string): PhaseBadgeData => {
  const foundVisibility = visibilityTypes.find((v) => v.value === visibility);
  if (foundVisibility) {
    return foundVisibility;
  } else {
    return {
      label: "未知",
      color: "text-highlight-slate bg-highlight-slate/20",
      description: "由于某些原因无法获取可见性信息",
    };
  }
};

const VisibilityBadge = ({ visibility }: { visibility: string }) => {
  return <PhaseBadge phase={visibility} getPhaseLabel={getVisibilityLabel} />;
};

export default VisibilityBadge;

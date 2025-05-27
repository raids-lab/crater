import { PhaseBadge, PhaseBadgeData } from "./PhaseBadge";

export enum Visibility {
  Public = "Public",
  Private = "Private",
  UserShare = "UserShare",
  AccountShare = "AccountShare",
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
    description: "仅限创建者可查看和使用的资源",
  },
  {
    value: Visibility.UserShare,
    label: "分享",
    color: "text-highlight-orange bg-highlight-orange/20",
    description: "其他用户分享的可查看和使用的资源",
  },
  {
    value: Visibility.AccountShare,
    label: "账户",
    color: "text-highlight-purple bg-highlight-purple/20",
    description: "同账户下的用户可查看和使用的资源",
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

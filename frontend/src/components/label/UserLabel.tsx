import { IUserInfo } from "@/services/api/vcjob";
import TooltipLink from "./TooltipLink";
import { useIsAdmin } from "@/hooks/useAdmin";
import { cn } from "@/lib/utils";
import { useAtomValue } from "jotai";
import { globalHideUsername } from "@/utils/store";
import { getUserPseudonym } from "@/utils/pseudonym";

const UserLabel = ({
  info,
  className,
}: {
  info: IUserInfo;
  className?: string;
}) => {
  const hideUsername = useAtomValue(globalHideUsername);
  const isAdminMode = useIsAdmin();
  const prefix = isAdminMode ? "admin/user" : "portal/user";

  // 根据 hideUsername 状态决定显示真实名称还是假名
  const displayName = hideUsername
    ? getUserPseudonym(info.username)
    : info.nickname || info.username;

  return (
    <TooltipLink
      name={
        <span className={cn("truncate text-sm font-normal", className)}>
          {displayName}
        </span>
      }
      to={`/${prefix}/${info.username}`}
      tooltip={
        <p>
          查看{displayName}
          <span className="mx-0.5 font-mono">
            (@{hideUsername ? getUserPseudonym(info.username) : info.username})
          </span>
          信息
        </p>
      }
    />
  );
};

export default UserLabel;

import { IUserInfo } from "@/services/api/vcjob";
import TooltipLink from "./TooltipLink";
import { useIsAdmin } from "@/hooks/useAdmin";
import { cn } from "@/lib/utils";

const UserLabel = ({
  info,
  className,
}: {
  info: IUserInfo;
  className?: string;
}) => {
  const isAdminMode = useIsAdmin();
  const prefix = isAdminMode ? "admin/user" : "portal/user";
  return (
    <TooltipLink
      name={
        <span className={cn("truncate text-sm font-normal", className)}>
          {info.nickname || info.username}
        </span>
      }
      to={`/${prefix}/${info.username}`}
      tooltip={
        <p>
          查看{info.nickname || info.username}
          <span className="mx-0.5 font-mono">(@{info.username})</span>
          信息
        </p>
      }
    />
  );
};

export default UserLabel;

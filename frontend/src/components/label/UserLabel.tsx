import { IUserInfo } from "@/services/api/vcjob";
import TooltipLink from "./TooltipLink";

const UserLabel = ({
  attributes,
  prefix,
}: {
  attributes: IUserInfo;
  prefix: string;
}) => {
  return (
    <TooltipLink
      name={
        <span className="truncate text-sm font-normal">
          {attributes.nickname || attributes.username}
        </span>
      }
      to={`/${prefix}/${attributes.username}`}
      tooltip={
        <p>
          查看{attributes.nickname || attributes.username}
          <span className="mx-0.5 font-mono">(@{attributes.username})</span>
          信息
        </p>
      }
    />
  );
};

export default UserLabel;

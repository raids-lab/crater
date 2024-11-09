import { IUserAttributes } from "@/services/api/admin/user";

const UserLabel = ({ attributes }: { attributes: IUserAttributes }) => {
  return (
    <div className="flex flex-col gap-0.5 text-left">
      <span className="truncate text-sm font-normal">
        {attributes.nickname ?? attributes.name}
      </span>
      <span className="truncate text-xs text-muted-foreground">
        @{attributes.name}
      </span>
    </div>
  );
};

export default UserLabel;

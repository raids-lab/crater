import Select, { OnChangeValue } from "react-select";
import { useQuery } from "@tanstack/react-query";
import { apiListUsersNotInDataset } from "@/services/api/dataset";
interface UserOption {
  value: string;
  id: number;
  label: string;
}
interface UserSelectProps {
  id: number;
  onChange: (newValue: OnChangeValue<UserOption, true>) => void;
}
export function UserNotInSelect({ id, onChange }: UserSelectProps) {
  const { data: userList } = useQuery({
    queryKey: ["dataset", "userOutList", { id }],
    queryFn: () => apiListUsersNotInDataset(id),
    select: (res) => {
      return res.data.data.map((user) => {
        return {
          value: user.name,
          label: user.name,
          id: user.id,
        };
      });
    },
  });

  return (
    <Select
      isMulti
      name="users"
      options={userList}
      className="basic-multi-select col-span-3"
      classNamePrefix="select"
      onChange={onChange}
    />
  );
}

import React from "react";
import { useQuery } from "@tanstack/react-query";
import { apiAdminAccountList } from "@/services/api/account";
import { IAccount } from "@/services/api/account";
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from "@/components/ui/select";

interface AccountSelectProps {
  value: string;
  onChange: (value: string) => void;
}

const AccountSelect: React.FC<AccountSelectProps> = ({ value, onChange }) => {
  const { data } = useQuery({
    queryKey: ["admin", "accounts"],
    queryFn: apiAdminAccountList,
  });

  return (
    <div>
      <Select value={value} onValueChange={onChange}>
        <SelectTrigger id="account-select">
          <SelectValue placeholder="请选择账户" />
        </SelectTrigger>
        <SelectContent>
          {data?.data?.data
            .filter((account: IAccount) => account.name !== "default")
            .map((account: IAccount) => (
              <SelectItem key={account.name} value={account.name.toString()}>
                {account.nickname}
              </SelectItem>
            ))}
        </SelectContent>
      </Select>
    </div>
  );
};

export default AccountSelect;

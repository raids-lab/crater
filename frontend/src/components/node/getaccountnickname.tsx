import { useQuery } from "@tanstack/react-query";
import { apiAdminAccountList } from "@/services/api/account";
import { IAccount } from "@/services/api/account";

/**
 * 获取账户昵称的工具函数
 */
export const useAccountNameLookup = () => {
  // 获取所有账户数据
  const { data, isLoading, error } = useQuery({
    queryKey: ["admin", "accounts"],
    queryFn: apiAdminAccountList,
  });

  const accounts = data?.data?.data || [];

  /**
   * 通过账户名获取对应的昵称
   * @param name 账户名
   * @returns 对应的昵称，如果未找到则返回undefined
   */
  const getNicknameByName = (name: string): string | undefined => {
    const account = accounts.find((acc: IAccount) => acc.name === name);
    return account?.nickname;
  };

  return {
    getNicknameByName,
    isLoading,
    error,
  };
};
export default useAccountNameLookup;

import { NavGroup } from "@/components/sidebar/nav-main";
import { NavUser } from "@/components/sidebar/nav-user";
import { TeamSwitcher } from "@/components/sidebar/team-switcher";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarRail,
} from "@/components/ui/sidebar";
import useIsAdmin from "@/hooks/useAdmin";
import { Role } from "@/services/api/auth";
import { globalAccount } from "@/utils/store";
import { useAtomValue } from "jotai";
import { UsersRoundIcon } from "lucide-react";
import { useMemo } from "react";
import { NavGroupProps } from "./types";

export function AppSidebar({
  groups,
  ...props
}: React.ComponentProps<typeof Sidebar> & {
  groups: NavGroupProps[];
}) {
  const isAdminView = useIsAdmin();
  const accountInfo = useAtomValue(globalAccount);

  // 特殊规则，当当前账户为其他账户时，且当前用户的权限是账户管理员，且当前处于用户模式时，添加账户管理菜单
  const filteredGroups = useMemo(() => {
    if (
      !isAdminView &&
      accountInfo.queue !== "default" &&
      accountInfo.roleQueue === Role.Admin &&
      groups.length > 0 &&
      groups[groups.length - 1].items.length > 0 &&
      groups[groups.length - 1].items[0].title !== "账户管理"
    ) {
      groups[groups.length - 1].items = [
        {
          title: "账户管理",
          icon: UsersRoundIcon,
          items: [
            {
              title: "成员管理",
              url: "account/member",
            },
          ],
        },
        ...groups[groups.length - 1].items,
      ];
      return groups;
    }
    // revert
    if (
      (isAdminView ||
        accountInfo.queue === "default" ||
        accountInfo.roleQueue !== Role.Admin) &&
      groups.length > 0 &&
      groups[groups.length - 1].items.length > 0 &&
      groups[groups.length - 1].items[0].title === "账户管理"
    ) {
      groups[groups.length - 1].items =
        groups[groups.length - 1].items.slice(1);
    }
    return groups;
  }, [isAdminView, accountInfo.queue, accountInfo.roleQueue, groups]);

  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarHeader>
        <TeamSwitcher />
      </SidebarHeader>
      <SidebarContent className="gap-0">
        {filteredGroups.map((group) => (
          <NavGroup key={group.title} {...group} />
        ))}
      </SidebarContent>
      <SidebarFooter>
        <NavUser />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  );
}

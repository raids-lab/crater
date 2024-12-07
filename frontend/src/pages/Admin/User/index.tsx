import { DotsHorizontalIcon } from "@radix-ui/react-icons";
import { ColumnDef } from "@tanstack/react-table";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui-custom/alert-dialog";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useMemo } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { showErrorToast } from "@/utils/toast";
import { toast } from "sonner";
import { DataTable } from "@/components/custom/DataTable";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import {
  apiAdminUserDelete,
  apiAdminUserList,
  apiAdminUserUpdateRole,
  IUserAttributes,
} from "@/services/api/admin/user";
import { useAtomValue } from "jotai";
import { globalUserInfo } from "@/utils/store";
import { Role } from "@/services/api/auth";
import { ProjectStatus } from "@/services/api/account";
import UserLabel from "@/components/label/UserLabel";
import UserRoleBadge from "@/components/label/UserRoleBadge";
import UserStatusBadge from "@/components/label/UserStatusBadge";

interface TUser {
  id: number;
  name: string;
  role: string;
  status: string;
  attributes: IUserAttributes;
}

const getHeader = (key: string): string => {
  switch (key) {
    case "name":
      return "用户";
    case "group":
      return "组别";
    case "teacher":
      return "导师";
    case "role":
      return "权限";
    case "status":
      return "状态";
    default:
      return key;
  }
};

const roles = [
  {
    label: "管理员",
    value: Role.Admin.toString(),
  },
  {
    label: "普通用户",
    value: Role.User.toString(),
  },
];

const statuses = [
  {
    label: "已激活",
    value: ProjectStatus.Active.toString(),
  },
  {
    label: "已禁用",
    value: ProjectStatus.Inactive.toString(),
  },
];

const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: "搜索用户名",
    key: "name",
  },
  filterOptions: [
    {
      key: "role",
      title: "权限",
      option: roles,
    },
    {
      key: "status",
      title: "状态",
      option: statuses,
    },
  ],
  getHeader: getHeader,
};

export const User = () => {
  const queryClient = useQueryClient();
  const { name: currentUserName } = useAtomValue(globalUserInfo);

  const userQuery = useQuery({
    queryKey: ["admin", "userlist"],
    queryFn: apiAdminUserList,
    select: (res) =>
      res.data.data.map((item) => ({
        id: item.id,
        name: item.name,
        role: item.role.toString(),
        status: item.status.toString(),
        attributes: item.attributes,
      })),
  });

  const { mutate: deleteUser } = useMutation({
    mutationFn: (userName: string) => apiAdminUserDelete(userName),
    onSuccess: async (_, userName) => {
      await queryClient.invalidateQueries({ queryKey: ["admin", "userlist"] });
      toast.success(`用户 ${userName} 已删除`);
    },
  });

  const { mutate: updateRole } = useMutation({
    mutationFn: ({ userName, role }: { userName: string; role: Role }) =>
      apiAdminUserUpdateRole(userName, role),
    onSuccess: async (_, variables) => {
      await queryClient.invalidateQueries({ queryKey: ["admin", "userlist"] });
      toast.success(`用户 ${variables.userName} 权限已更新`);
    },
  });

  const columns = useMemo<ColumnDef<TUser>[]>(
    () => [
      {
        accessorKey: "name",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("name")} />
        ),
        cell: ({ row }) => <UserLabel attributes={row.original.attributes} />,
      },
      {
        accessorKey: "group",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("group")} />
        ),
        cell: ({ row }) => <div>{row.original.attributes.group}</div>,
      },
      {
        accessorKey: "teacher",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("teacher")} />
        ),
        cell: ({ row }) => <div>{row.original.attributes.teacher}</div>,
      },

      {
        accessorKey: "role",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("role")} />
        ),
        cell: ({ row }) => {
          return <UserRoleBadge role={row.getValue("role")} />;
        },
        filterFn: (row, id, value) => {
          return (value as string[]).includes(row.getValue(id));
        },
      },
      {
        accessorKey: "status",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("status")} />
        ),
        cell: ({ row }) => {
          return <UserStatusBadge status={row.getValue("status")} />;
        },
        filterFn: (row, id, value) => {
          return (value as string[]).includes(row.getValue(id));
        },
      },
      {
        id: "actions",
        enableHiding: false,
        cell: ({ row }) => {
          const user = row.original;
          return (
            <div>
              <AlertDialog>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button
                      variant="ghost"
                      className="h-8 w-8 p-0"
                      title="更多选项"
                    >
                      <DotsHorizontalIcon className="h-4 w-4" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuLabel className="text-xs text-muted-foreground">
                      操作
                    </DropdownMenuLabel>
                    <DropdownMenuSub>
                      <DropdownMenuSubTrigger>权限</DropdownMenuSubTrigger>
                      <DropdownMenuSubContent>
                        <DropdownMenuRadioGroup value={`${user.role}`}>
                          {roles.map((role) => (
                            <DropdownMenuRadioItem
                              key={role.value}
                              value={`${role.value}`}
                              onClick={() =>
                                updateRole({
                                  userName: user.name,
                                  role: parseInt(role.value),
                                })
                              }
                            >
                              {role.label}
                            </DropdownMenuRadioItem>
                          ))}
                        </DropdownMenuRadioGroup>
                      </DropdownMenuSubContent>
                    </DropdownMenuSub>
                    <DropdownMenuSeparator />
                    <AlertDialogTrigger asChild>
                      <DropdownMenuItem className="focus:bg-destructive focus:text-destructive-foreground">
                        删除
                      </DropdownMenuItem>
                    </AlertDialogTrigger>
                  </DropdownMenuContent>
                </DropdownMenu>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>删除用户</AlertDialogTitle>
                    <AlertDialogDescription>
                      用户「{user?.name}」将被删除，请谨慎操作。
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>取消</AlertDialogCancel>
                    <AlertDialogAction
                      variant="destructive"
                      onClick={() => {
                        if (user.name === currentUserName) {
                          showErrorToast(
                            "无法删除自己，如需删除请换个用户登录",
                          );
                        } else {
                          deleteUser(user.name);
                        }
                      }}
                    >
                      删除
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            </div>
          );
        },
      },
    ],
    [deleteUser, currentUserName, updateRole],
  );

  return (
    <DataTable
      query={userQuery}
      columns={columns}
      toolbarConfig={toolbarConfig}
    />
  );
};

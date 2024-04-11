import {
  DotsHorizontalIcon,
  StarFilledIcon,
  StarIcon,
} from "@radix-ui/react-icons";
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
import { Checkbox } from "@/components/ui/checkbox";
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
import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { showErrorToast } from "@/utils/toast";
import { toast } from "sonner";
import { DataTable } from "@/components/custom/OldDataTable";
import { DataTableColumnHeader } from "@/components/custom/OldDataTable/DataTableColumnHeader";
import { DataTableToolbarConfig } from "@/components/custom/OldDataTable/DataTableToolbar";
import {
  apiAdminUserDelete,
  apiAdminUserList,
  apiAdminUserUpdateRole,
} from "@/services/api/admin/user";
import { useRecoilValue } from "recoil";
import { globalUserInfo } from "@/utils/store";
import { Role } from "@/services/api/auth";
import { ProjectStatus } from "@/services/api/project";

interface TUser {
  id: number;
  name: string;
  role: Role;
  status: ProjectStatus;
}

const getHeader = (key: string): string => {
  switch (key) {
    case "userName":
      return "用户名";
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
    value: Role.Admin,
    icon: StarFilledIcon,
  },
  {
    label: "普通用户",
    value: Role.User,
    icon: StarIcon,
  },
];

const statuses = [
  {
    label: "已激活",
    value: ProjectStatus.Active,
    icon: StarFilledIcon,
  },
  {
    label: "未激活",
    value: ProjectStatus.Inactive,
    icon: StarIcon,
  },
];

const toolbarConfig: DataTableToolbarConfig<Role> = {
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
  ],
  getHeader: getHeader,
};

export const User = () => {
  const [data, setData] = useState<TUser[]>([]);
  const queryClient = useQueryClient();
  const { name: currentUserName } = useRecoilValue(globalUserInfo);

  const { data: userList, isLoading } = useQuery({
    queryKey: ["admin", "userlist"],
    queryFn: apiAdminUserList,
    select: (res) => res.data.data,
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
        id: "select",
        header: ({ table }) => (
          <Checkbox
            checked={table.getIsAllPageRowsSelected()}
            onCheckedChange={(value) =>
              table.toggleAllPageRowsSelected(!!value)
            }
            aria-label="Select all"
          />
        ),
        cell: ({ row }) => (
          <Checkbox
            checked={row.getIsSelected()}
            onCheckedChange={(value) => row.toggleSelected(!!value)}
            aria-label="Select row"
          />
        ),
        enableSorting: false,
        enableHiding: false,
      },
      {
        accessorKey: "name",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("name")} />
        ),
        cell: ({ row }) => <div>{row.getValue("name")}</div>,
      },
      {
        accessorKey: "role",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("role")} />
        ),
        cell: ({ row }) => {
          const role = roles.find(
            (role) => role.value === row.getValue("role"),
          );
          if (!role) {
            return null;
          }
          return (
            <div className="flex items-center">
              {role.icon && (
                <role.icon className="mr-2 h-4 w-4 text-muted-foreground" />
              )}
              <span>{role.label}</span>
            </div>
          );
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
          const status = statuses.find(
            (status) => status.value === row.getValue("status"),
          );
          if (!status) {
            return null;
          }
          return (
            <div className="flex items-center">
              {status.icon && (
                <status.icon className="mr-2 h-4 w-4 text-muted-foreground" />
              )}
              <span>{status.label}</span>
            </div>
          );
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
                    <DropdownMenuLabel>操作</DropdownMenuLabel>
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
                                  role: role.value,
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
                      <DropdownMenuItem>删除</DropdownMenuItem>
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

  useEffect(() => {
    if (isLoading) return;
    if (!userList) return;
    const tableData: TUser[] = userList.map((user) => ({
      id: user.id,
      name: user.name,
      role: user.role,
      status: user.status,
    }));
    setData(tableData);
  }, [userList, isLoading]);

  return (
    <DataTable
      data={data}
      columns={columns}
      toolbarConfig={toolbarConfig}
      loading={isLoading}
    />
  );
};

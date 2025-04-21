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
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
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
import { DataTable } from "@/components/custom/DataTable";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import {
  apiAdminUserDelete,
  apiAdminUserList,
  apiAdminUpdateUserAttributes,
  apiAdminUserUpdateRole,
  IUserAttributes,
} from "@/services/api/admin/user";
import { useAtomValue } from "jotai";
import { globalUserInfo } from "@/utils/store";
import { Role } from "@/services/api/auth";
import { ProjectStatus } from "@/services/api/account";
import UserLabel from "@/components/label/UserLabel";
import UserRoleBadge from "@/components/badge/UserRoleBadge";
import UserStatusBadge from "@/components/badge/UserStatusBadge";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";

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

const userFormSchema = z.object({
  nickname: z.string().optional(),
  email: z.string().email().optional().or(z.literal("")),
  teacher: z.string().optional().or(z.literal("")),
  group: z.string().optional().or(z.literal("")),
  phone: z.string().optional().or(z.literal("")),
});

type UserFormValues = z.infer<typeof userFormSchema>;

interface UserEditDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  user: TUser | null;
}

function UserEditDialog({ open, onOpenChange, user }: UserEditDialogProps) {
  const queryClient = useQueryClient();

  const form = useForm<UserFormValues>({
    resolver: zodResolver(userFormSchema),
    defaultValues: {
      nickname: user?.attributes.nickname || "",
      email: user?.attributes.email || "",
      teacher: user?.attributes.teacher || "",
      group: user?.attributes.group || "",
      phone: user?.attributes.phone || "",
    },
  });

  useEffect(() => {
    if (user) {
      form.reset({
        nickname: user.attributes.nickname || "",
        email: user.attributes.email || "",
        teacher: user.attributes.teacher || "",
        group: user.attributes.group || "",
        phone: user.attributes.phone || "",
      });
    }
  }, [form, user]);

  const { mutate: updateUser, isPending } = useMutation({
    mutationFn: (values: UserFormValues) => {
      if (!user) throw new Error("No user selected");
      const updateData: IUserAttributes = {
        ...user.attributes,
        ...values,
      };
      return apiAdminUpdateUserAttributes(user.name, updateData);
    },
    onSuccess: () => {
      toast.success("用户信息已更新");
      queryClient.invalidateQueries({ queryKey: ["admin", "userlist"] });
      onOpenChange(false);
    },
    onError: () => {
      toast.error("更新失败");
    },
  });

  function onSubmit(values: UserFormValues) {
    updateUser(values);
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>编辑用户信息</DialogTitle>
          <DialogDescription>
            更新用户 {user?.name} 的个人信息
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="nickname"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>昵称</FormLabel>
                  <FormControl>
                    <Input placeholder="昵称" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="email"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>邮箱</FormLabel>
                  <FormControl>
                    <Input placeholder="邮箱地址" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="teacher"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>导师</FormLabel>
                  <FormControl>
                    <Input placeholder="导师姓名" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="group"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>组别</FormLabel>
                  <FormControl>
                    <Input placeholder="组别" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="phone"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>手机</FormLabel>
                  <FormControl>
                    <Input placeholder="手机号码" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <DialogFooter>
              <Button type="submit" disabled={isPending}>
                {isPending ? "保存中..." : "保存修改"}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}

export const User = () => {
  const queryClient = useQueryClient();
  const { name: currentUserName } = useAtomValue(globalUserInfo);
  const [editUser, setEditUser] = useState<TUser | null>(null);
  const [editDialogOpen, setEditDialogOpen] = useState(false);

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
        cell: ({ row }) => (
          <UserLabel
            info={{
              username: row.original.name,
              nickname: row.original.attributes.nickname,
            }}
          />
        ),
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
                      <DotsHorizontalIcon className="size-4" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuLabel className="text-muted-foreground text-xs">
                      操作
                    </DropdownMenuLabel>
                    <DropdownMenuItem
                      onClick={() => {
                        setEditUser(user);
                        setEditDialogOpen(true);
                      }}
                    >
                      编辑信息
                    </DropdownMenuItem>
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
    <>
      <DataTable
        info={{
          title: "用户管理",
          description: "在这里查看和管理用户信息",
        }}
        storageKey="admin_user"
        query={userQuery}
        columns={columns}
        toolbarConfig={toolbarConfig}
      />
      <UserEditDialog
        open={editDialogOpen}
        onOpenChange={setEditDialogOpen}
        user={editUser}
      />
    </>
  );
};

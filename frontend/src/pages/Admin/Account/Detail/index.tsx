import { Button } from "@/components/ui/button";
import { DotsHorizontalIcon } from "@radix-ui/react-icons";
import {
  Form,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormControl,
} from "@/components/ui/form";
import FormLabelMust from "@/components/form/FormLabelMust";
import { DialogTrigger, Dialog, DialogClose } from "@/components/ui/dialog";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import {
  SelectValue,
  SelectTrigger,
  SelectItem,
  SelectContent,
  Select,
  SelectGroup,
} from "@/components/ui/select";
import {
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogHeader,
  AlertDialogCancel,
  AlertDialogAction,
  AlertDialogFooter,
  AlertDialogContent,
  AlertDialog,
  AlertDialogTrigger,
} from "@/components/ui-custom/alert-dialog";
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
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo, useState } from "react";
import { useParams } from "react-router-dom";
import {
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogContent,
} from "@/components/ui/dialog";
import {
  IUserInAccount,
  apiAddUser,
  apiUpdateUser,
  apiRemoveUser,
  apiUserInProjectList,
  apiUserOutOfProjectList,
  Access,
  IUserInAccountCreate,
  apiAccountGet,
} from "@/services/api/account";
import { ColumnDef } from "@tanstack/react-table";
import { DataTable } from "@/components/custom/DataTable";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import { z } from "zod";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import UserRoleBadge, { userRoles } from "@/components/badge/UserRoleBadge";
import UserAccessBadge from "@/components/badge/UserAccessBadge";
import ResourceBadges from "@/components/badge/ResourceBadges";
import {
  Briefcase,
  Calendar,
  Layers,
  UserRoundPlusIcon,
  Users,
} from "lucide-react";
import Quota from "./AccountQuota";
import { toast } from "sonner";
import SelectBox from "@/components/custom/SelectBox";
import UserLabel from "@/components/label/UserLabel";
import { DetailPage } from "@/components/layout/DetailPage";
import { TimeDistance } from "@/components/custom/TimeDistance";
import { Skeleton } from "@/components/ui/skeleton";
import DetailTitle from "@/components/layout/DetailTitle";

const formSchema = z.object({
  userIds: z.array(z.string()).min(1, {
    message: "请至少选择一个用户",
  }),
  role: z.string().min(1, {
    message: "不支持的角色",
  }),
  accessmode: z.string().min(1, {
    message: "不支持的访问权限",
  }),
});

const getHeader = (key: string): string => {
  switch (key) {
    case "name":
      return "用户名";
    case "role":
      return "账户权限";
    case "accessmode":
      return "读写权限";
    case "capability":
      return "资源上限";
    default:
      return key;
  }
};

const accessmodes = [
  {
    label: "只读",
    value: Access.RO.toString(),
  },
  {
    label: "读写",
    value: Access.RW.toString(),
  },
];

const AccountDetail = () => {
  const { id } = useParams();
  const queryClient = useQueryClient();
  const pid = useMemo(() => Number(id), [id]);

  const setBreadcrumb = useBreadcrumb();

  const accountUsersQuery = useQuery({
    queryKey: ["account", pid, "users"],
    queryFn: () => apiUserInProjectList(pid),
    select: (res) => res.data.data,
  });

  const { data: accountInfo, isLoading: isLoadingAccount } = useQuery({
    queryKey: ["admin", "accounts", pid],
    queryFn: () => apiAccountGet(pid),
    select: (res) => res.data.data,
    enabled: !!pid,
  });

  // 修改 BreadCrumb
  useEffect(() => {
    if (!accountInfo) return;
    setBreadcrumb([{ title: accountInfo?.nickname }]);
  }, [setBreadcrumb, accountInfo]);

  const { mutate: addUser } = useMutation({
    mutationFn: (users: IUserInAccountCreate[]) =>
      Promise.all(users.map((user) => apiAddUser(pid, user))),
    onSuccess: async () => {
      toast.success("用户已添加");
      await queryClient.invalidateQueries({
        queryKey: ["account", pid, "users"],
      });
    },
  });

  const { mutate: updateUser } = useMutation({
    mutationFn: (user: IUserInAccountCreate) => apiUpdateUser(pid, user),
    onSuccess: async () => {
      toast.success("用户信息已更新");
      await queryClient.invalidateQueries({
        queryKey: ["account", pid, "users"],
      });
    },
  });

  const [usersOutOfProject, setUsersOutOfProject] = useState<IUserInAccount[]>(
    [],
  );

  const { mutate: deleteUser } = useMutation({
    mutationFn: (user: IUserInAccountCreate) => apiRemoveUser(pid, user),
    onSuccess: async () => {
      toast.success("用户已删除");
      await queryClient.invalidateQueries({
        queryKey: ["account", pid, "users"],
      });
    },
  });

  const { data: usersOutOfProjectData, isLoading: isLoadingUsersOutOfProject } =
    useQuery({
      queryKey: ["usersOutOfProject", pid],
      queryFn: () => apiUserOutOfProjectList(pid),
      select: (res) => res.data.data,
    });

  useEffect(() => {
    if (isLoadingUsersOutOfProject) return;
    if (!usersOutOfProjectData) return;
    setUsersOutOfProject(usersOutOfProjectData);
  }, [usersOutOfProjectData, isLoadingUsersOutOfProject]);

  const columns = useMemo<ColumnDef<IUserInAccount>[]>(
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
              nickname: row.original.userInfo.nickname,
            }}
          />
        ),
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
        accessorKey: "accessmode",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("accessmode")}
          />
        ),
        cell: ({ row }) => (
          <UserAccessBadge access={row.getValue("accessmode")} />
        ),
        filterFn: (row, id, value) => {
          return (value as string[]).includes(row.getValue(id));
        },
      },
      {
        accessorKey: "capability",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("capability")}
          />
        ),
        cell: ({ row }) => {
          return <ResourceBadges resources={row.original.quota.capability} />;
        },
        enableSorting: false,
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
                    <DropdownMenuSub>
                      <DropdownMenuSubTrigger>角色</DropdownMenuSubTrigger>
                      <DropdownMenuSubContent>
                        <DropdownMenuRadioGroup value={`${user.role}`}>
                          {userRoles.map((role) => (
                            <DropdownMenuRadioItem
                              key={role.value}
                              value={`${role.value}`}
                              onClick={() =>
                                updateUser({
                                  id: user.id,
                                  name: user.name,
                                  role: role.value,
                                  accessmode: user.accessmode,
                                })
                              }
                            >
                              {role.label}
                            </DropdownMenuRadioItem>
                          ))}
                        </DropdownMenuRadioGroup>
                      </DropdownMenuSubContent>
                    </DropdownMenuSub>
                    <DropdownMenuSub>
                      <DropdownMenuSubTrigger>权限</DropdownMenuSubTrigger>
                      <DropdownMenuSubContent>
                        <DropdownMenuRadioGroup value={`${user.accessmode}`}>
                          {accessmodes.map((accessmode) => (
                            <DropdownMenuRadioItem
                              key={accessmode.value}
                              value={`${accessmode.value}`}
                              onClick={() =>
                                updateUser({
                                  id: user.id,
                                  name: user.name,
                                  role: user.role,
                                  accessmode: accessmode.value,
                                })
                              }
                            >
                              {accessmode.label}
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
                        deleteUser(user);
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
    [deleteUser, updateUser],
  );

  const toolbarConfig: DataTableToolbarConfig = {
    filterInput: {
      placeholder: "搜索用户名",
      key: "name",
    },
    filterOptions: [
      {
        key: "role",
        title: "角色",
        option: userRoles,
      },
      {
        key: "accessmode",
        title: "权限",
        option: accessmodes,
      },
    ],
    getHeader: getHeader,
  };

  const [openSheet, setOpenSheet] = useState(false);

  // 1. Define your form.
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      userIds: [],
      role: "2",
      accessmode: "2",
    },
  });

  // 2. Define a submit handler.
  const onSubmit = (values: z.infer<typeof formSchema>) => {
    // 处理多个用户添加
    const usersToAdd = values.userIds.map((userId) => {
      const user = usersOutOfProject.find((u) => u.id.toString() === userId);
      return {
        id: user?.id as number,
        name: user?.name as string,
        role: values.role,
        accessmode: values.accessmode,
        attributes: user?.userInfo,
      };
    });

    addUser(usersToAdd);
    setOpenSheet(false);
  };

  // 将用户列表转换为SelectBox需要的格式
  const userOptions = useMemo(
    () =>
      usersOutOfProject.map((user) => ({
        value: user.id.toString(),
        label: user.userInfo.nickname || user.name,
        labelNote: user.name,
      })),
    [usersOutOfProject],
  );

  // 加载中状态
  if (isLoadingAccount) {
    return (
      <DetailPage
        header={
          <div className="flex items-center space-x-4">
            <Skeleton className="h-12 w-12 rounded-full" />
            <div>
              <Skeleton className="mb-2 h-8 w-40" />
              <Skeleton className="h-4 w-20" />
            </div>
          </div>
        }
        info={[]}
        tabs={[]}
      />
    );
  }

  // 错误状态或数据不存在
  if (!accountInfo) {
    return (
      <DetailPage
        header={
          <div>
            <h1 className="text-2xl font-bold text-red-500">
              无法加载账户数据
            </h1>
            <p className="text-muted-foreground">账户不存在或您没有权限访问</p>
          </div>
        }
        info={[]}
        tabs={[]}
      />
    );
  }

  // 账户头部内容
  const header = (
    <DetailTitle
      icon={Briefcase}
      title={accountInfo.nickname}
      description={accountInfo.name}
    />
  );

  // 账户基本信息
  const info = [
    {
      icon: Users,
      title: "用户数量",
      value: accountUsersQuery.data?.length || "加载中...",
    },
    {
      icon: Layers,
      title: "账户级别",
      value: "标准",
    },
    {
      icon: Calendar,
      title: "过期时间",
      value: <TimeDistance date={accountInfo.expiredAt} />,
    },
  ];

  // 用户管理组件
  const UserManagement = () => (
    <DataTable
      storageKey="admin_account_users"
      query={accountUsersQuery}
      columns={columns}
      toolbarConfig={toolbarConfig}
    >
      <Dialog open={openSheet} onOpenChange={setOpenSheet}>
        <DialogTrigger asChild>
          <Button className="h-8">
            <UserRoundPlusIcon className="size-4" />
            添加用户
          </Button>
        </DialogTrigger>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>添加用户</DialogTitle>
          </DialogHeader>
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
              <FormField
                control={form.control}
                name="userIds"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      用户
                      <FormLabelMust />
                    </FormLabel>
                    <FormControl>
                      <SelectBox
                        options={userOptions}
                        value={field.value}
                        onChange={field.onChange}
                        placeholder="选择用户"
                        inputPlaceholder="搜索用户..."
                        emptyPlaceholder="没有找到用户"
                      />
                    </FormControl>
                    <FormDescription>
                      可选择一位或多位用户加入到账户中
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="role"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      角色
                      <FormLabelMust />
                    </FormLabel>
                    <Select
                      onValueChange={field.onChange}
                      defaultValue={field.value.toString()}
                    >
                      <FormControl>
                        <SelectTrigger className="w-full" id="role">
                          <SelectValue placeholder="Select users" />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectGroup>
                          <SelectItem value="2">普通用户</SelectItem>
                          <SelectItem value="3">管理员</SelectItem>
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                    <FormDescription>配置用户在账户中的角色</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="accessmode"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      访问权限
                      <FormLabelMust />
                    </FormLabel>
                    <Select
                      onValueChange={field.onChange}
                      defaultValue={field.value.toString()}
                    >
                      <FormControl>
                        <SelectTrigger className="w-full" id="accessmode">
                          <SelectValue placeholder="Select users" />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectGroup>
                          <SelectItem value="2">只读</SelectItem>
                          <SelectItem value="3">读写</SelectItem>
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                    <FormDescription>
                      设置用户在账户空间的访问权限
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <DialogFooter>
                <DialogClose asChild>
                  <Button variant="outline">取消</Button>
                </DialogClose>
                <Button type="submit">添加</Button>
              </DialogFooter>
            </form>
          </Form>
        </DialogContent>
      </Dialog>
    </DataTable>
  );

  // 标签页配置
  const tabs = [
    {
      key: "users",
      icon: Users,
      label: "用户管理",
      children: <UserManagement />,
      scrollable: true,
    },
    {
      key: "quota",
      icon: Layers,
      label: "配额使用情况",
      children: (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          <Quota accountID={pid} />
        </div>
      ),
      scrollable: true,
    },
  ];

  return <DetailPage header={header} info={info} tabs={tabs} />;
};

export default AccountDetail;

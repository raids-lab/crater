import { DataTableColumnHeader } from "@/components/custom/PagenationDataTable/DataTableColumnHeader";
import { apiAdminProjectList } from "@/services/api/admin/user";
import { ProjectStatus } from "@/services/api/project";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { DotsHorizontalIcon } from "@radix-ui/react-icons";
import { Badge } from "@/components/ui/badge";
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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useEffect, useMemo, useState } from "react";
import {
  ColumnDef,
  ColumnFiltersState,
  OnChangeFn,
  PaginationState,
  SortingState,
  VisibilityState,
  getCoreRowModel,
  getFacetedRowModel,
  getFacetedUniqueValues,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
} from "@tanstack/react-table";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { logger } from "@/utils/loglevel";
import FormLabelMust from "@/components/custom/FormLabelMust";
import LoadableButton from "@/components/custom/LoadableButton";
import { useMutation } from "@tanstack/react-query";
import {
  DataTableToolbarConfig,
  DataTableToolbarRight,
} from "@/components/custom/PagenationDataTable/DataTableToolbar";
import DataTableCard from "@/components/custom/PagenationDataTable/DataTableCard";
import { Tabs } from "@/components/ui/tabs";
import { z } from "zod";
import { toast } from "sonner";
import { apiProjectCreate, apiProjectDelete } from "@/services/api/project";
import { useNavigate } from "react-router-dom";
import { PlusCircleIcon } from "lucide-react";

interface Resource {
  [key: string]: string;
}
interface DataTableProps<TData, TValue>
  extends React.HTMLAttributes<HTMLDivElement> {
  setStatus: (status: ProjectStatus) => void;
  columns: ColumnDef<TData, TValue>[];
  data: TData[];
  toolbarConfig: DataTableToolbarConfig;
  loading: boolean;
  onPaginationChange: OnChangeFn<PaginationState>;
  rowCount?: number;
  pagination: PaginationState;
  className?: string;
}

function TableWithTabs<TData, TValue>({
  setStatus,
  columns,
  data,
  toolbarConfig,
  loading,
  onPaginationChange,
  rowCount,
  pagination,
}: DataTableProps<TData, TValue>) {
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);
  const [sorting, setSorting] = useState<SortingState>([]);
  const [showNewTeamDialog, setShowNewTeamDialog] = useState(false);
  const [activeTab, setActiveTab] = useState<string>("active");
  const [cachedRowCount, setCachedRowCount] = useState(rowCount);
  const queryClient = useQueryClient();
  useEffect(() => {
    if (rowCount && rowCount !== cachedRowCount) setCachedRowCount(rowCount);
  }, [rowCount, cachedRowCount]);

  useEffect(() => {
    if (activeTab === "active") {
      setStatus(ProjectStatus.Active);
    } else if (activeTab === "pending") {
      setStatus(ProjectStatus.Pending);
    } else if (activeTab === "inactive") {
      setStatus(ProjectStatus.Inactive);
    }
  }, [activeTab, setStatus]);

  const { mutate: createNewProject, isPending } = useMutation({
    mutationFn: (values: z.infer<typeof formSchema>) =>
      apiProjectCreate({
        name: values.name,
        quota: JSON.stringify({
          cpu: values.cpu,
          gpu: values.gpu,
          memory: values.memory,
          storage: values.storage,
        }),
      }),
    onSuccess: async (_, { name }) => {
      await queryClient.invalidateQueries({
        queryKey: [
          "admin",
          "projects",
          false,
          2,
          pagination.pageIndex,
          pagination.pageSize,
        ],
      });
      toast.success(`账户 ${name} 创建成功`);
      setShowNewTeamDialog(false);
      form.reset();
    },
  });
  const table = useReactTable({
    data,
    columns,
    state: {
      sorting,
      columnVisibility,
      columnFilters,
      pagination,
    },
    // pagination
    manualPagination: true,
    onPaginationChange,
    rowCount: cachedRowCount,
    // shadcn example props
    enableRowSelection: true,
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    onColumnVisibilityChange: setColumnVisibility,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFacetedRowModel: getFacetedRowModel(),
    getFacetedUniqueValues: getFacetedUniqueValues(),
  });

  const formSchema = z.object({
    name: z
      .string()
      .min(1, {
        message: "账户名称不能为空",
      })
      .max(16, {
        message: "账户名称最多16个字符",
      }),
    cpu: z.number().int().min(0, {
      message: "如需取消上限，请联系管理员",
    }),
    gpu: z.number().int().min(0, {
      message: "如需取消上限，请联系管理员",
    }),
    memory: z.number().int().min(0, {
      message: "如需取消上限，请联系管理员",
    }),
    storage: z.number().int().min(0, {
      message: "如需取消上限，请联系管理员",
    }),
  });

  // 1. Define your form.
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: "",
      cpu: 0,
      gpu: 0,
      memory: 0,
      storage: 0,
    },
  });

  // 2. Define a submit handler.
  const onSubmit = (values: z.infer<typeof formSchema>) => {
    // Do something with the form values.
    // ✅ This will be type-safe and validated.
    !isPending && createNewProject(values);
  };

  return (
    <div className="flex flex-col gap-2 lg:col-span-3">
      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <div className="flex flex-row justify-between gap-2">
          <div>
            <Dialog
              open={showNewTeamDialog}
              onOpenChange={setShowNewTeamDialog}
            >
              <DialogTrigger asChild>
                <Button
                  className="h-8"
                  onSelect={() => {
                    setShowNewTeamDialog(true);
                  }}
                >
                  <PlusCircleIcon className="-ml-0.5 mr-2 h-4 w-4" />
                  新建账户
                </Button>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>新建账户</DialogTitle>
                  <DialogDescription>
                    账户可包含多名成员，成员间共享配额、镜像、数据等资源。
                  </DialogDescription>
                </DialogHeader>
                <Form {...form}>
                  <form
                    // eslint-disable-next-line @typescript-eslint/no-misused-promises
                    onSubmit={form.handleSubmit(onSubmit)}
                    className="grid gap-4 md:grid-cols-2"
                  >
                    <FormField
                      control={form.control}
                      name="name"
                      render={({ field }) => (
                        <FormItem className="col-span-2">
                          <FormLabel>
                            账户名称
                            <FormLabelMust />
                          </FormLabel>
                          <FormControl>
                            <Input autoComplete="off" {...field} />
                          </FormControl>
                          <FormDescription>
                            名称是账户的唯一标识，最多16个字符
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name="cpu"
                      render={() => (
                        <FormItem>
                          <FormLabel>
                            申请 CPU 核心数
                            <FormLabelMust />
                          </FormLabel>
                          <FormControl>
                            <Input
                              type="number"
                              {...form.register("cpu", { valueAsNumber: true })}
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name="gpu"
                      render={() => (
                        <FormItem>
                          <FormLabel>
                            申请 GPU 卡数
                            <FormLabelMust />
                          </FormLabel>
                          <FormControl>
                            <Input
                              type="number"
                              {...form.register("gpu", { valueAsNumber: true })}
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name="memory"
                      render={() => (
                        <FormItem>
                          <FormLabel>
                            内存上限 (GB)
                            <FormLabelMust />
                          </FormLabel>
                          <FormControl>
                            <Input
                              type="number"
                              {...form.register("memory", {
                                valueAsNumber: true,
                              })}
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name="storage"
                      render={() => (
                        <FormItem>
                          <FormLabel>
                            账户存储空间 (GB)
                            <FormLabelMust />
                          </FormLabel>
                          <FormControl>
                            <Input
                              type="number"
                              {...form.register("storage", {
                                valueAsNumber: true,
                              })}
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </form>
                </Form>
                <DialogFooter>
                  <LoadableButton
                    isLoading={isPending}
                    type="submit"
                    onClick={() => {
                      form
                        .trigger()
                        .then(() => {
                          if (form.formState.isValid) {
                            onSubmit(form.getValues());
                          }
                        })
                        .catch((e) => logger.debug(e));
                    }}
                  >
                    提交申请
                  </LoadableButton>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </div>
          <DataTableToolbarRight table={table} config={toolbarConfig} />
        </div>
      </Tabs>
      <DataTableCard table={table} loading={loading} columns={columns} />
    </div>
  );
}

type Project = {
  id: number;
  name: string;
  deserved: Resource;
};

const getHeader = (key: string): string => {
  switch (key) {
    case "id":
      return "ID";
    case "name":
      return "账户名称";
    case "deserved":
      return "配额";
    default:
      return key;
  }
};

const toolbarConfig: DataTableToolbarConfig = {
  searchKey: "name",
  filterOptions: [],
  getHeader: getHeader,
};

export const Project = ({ isPersonal }: { isPersonal: boolean }) => {
  const [pagination, setPagination] = useState<PaginationState>({
    pageSize: 10,
    pageIndex: 0,
  });
  const navigate = useNavigate();
  const [data, setData] = useState<Project[]>([]);
  const [status, setStatus] = useState<ProjectStatus>(ProjectStatus.Active);
  const queryClient = useQueryClient();
  const { data: projects, isLoading } = useQuery({
    queryKey: [
      "admin",
      "projects",
      isPersonal,
      status,
      pagination.pageIndex,
      pagination.pageSize,
    ],
    queryFn: () =>
      apiAdminProjectList({
        pageIndex: pagination.pageIndex,
        pageSize: pagination.pageSize,
        isPersonal,
        status,
      }),
    select: (res) => res.data.data,
  });

  const { mutate: deleteProject } = useMutation({
    mutationFn: (projectId: string) => apiProjectDelete({ id: projectId }),
    onSuccess: async (_, projectName) => {
      await queryClient.invalidateQueries({
        queryKey: [
          "admin",
          "projects",
          isPersonal,
          status,
          pagination.pageIndex,
          pagination.pageSize,
        ],
      });
      toast.success(`账户 ${projectName} 已删除`);
    },
  });

  const columns = useMemo<ColumnDef<Project>[]>(
    () => [
      {
        accessorKey: "id",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("id")} />
        ),
        cell: ({ row }) => <div>{row.getValue("id")}</div>,
        enableSorting: false,
      },
      {
        accessorKey: "name",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("name")} />
        ),
        cell: ({ row }) => <div>{row.getValue("name")}</div>,
        enableSorting: false,
      },
      {
        accessorKey: "deserved",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("deserved")}
          />
        ),
        cell: ({ row }) => {
          const quota = row.getValue<Project["deserved"]>("deserved");
          return (
            <div className="grid grid-cols-3 gap-1 rounded-md border p-3 text-xs">
              {quota &&
                Object.entries(quota).map(([key, value]) => (
                  <Badge
                    key={key}
                    variant="secondary"
                    className="gap-2 font-medium"
                  >
                    {key}: {value}
                  </Badge>
                ))}
            </div>
          );
        },
        enableSorting: false,
      },
      {
        id: "actions",
        enableHiding: false,
        cell: ({ row }) => {
          const proj = row.original;
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
                    <DropdownMenuSeparator />
                    <DropdownMenuItem onClick={() => navigate(`${proj.id}`)}>
                      管理用户
                    </DropdownMenuItem>
                    <AlertDialogTrigger asChild>
                      <DropdownMenuItem>删除</DropdownMenuItem>
                    </AlertDialogTrigger>
                  </DropdownMenuContent>
                </DropdownMenu>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>删除项目</AlertDialogTitle>
                    <AlertDialogDescription>
                      账户 {proj?.name} 将被删除，请谨慎操作。
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>取消</AlertDialogCancel>
                    <AlertDialogAction
                      variant="destructive"
                      onClick={() => {
                        deleteProject(String(proj.id));
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
    [deleteProject, navigate],
  );

  useEffect(() => {
    if (isLoading) return;
    if (!projects?.rows) return;
    const tableData: Project[] = projects.rows
      //.filter((task) => !task.isDeleted)
      .map((p) => {
        return {
          id: p.ID,
          name: p.Nickname,
          deserved: JSON.parse(p.deserved) as Resource,
        };
      });
    setData(tableData);
  }, [projects, isLoading]);

  return (
    <TableWithTabs
      data={data}
      columns={columns}
      toolbarConfig={toolbarConfig}
      setStatus={setStatus}
      loading={isLoading}
      pagination={pagination}
      onPaginationChange={setPagination}
      rowCount={projects?.count || 0}
    />
  );
};

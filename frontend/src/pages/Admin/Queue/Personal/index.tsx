import { DataTableColumnHeader } from "@/components/custom/PagenationDataTable/DataTableColumnHeader";
import { apiAdminProjectList } from "@/services/api/admin/user";
import { ProjectStatus, apiProjectUpdate } from "@/services/api/project";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { CirclePlusIcon, XIcon } from "lucide-react";
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
import { useFieldArray, useForm } from "react-hook-form";
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
import { apiResourceList } from "@/services/api/resource";
import { ScrollArea, ScrollBar } from "@/components/ui/scroll-area";
import useResizeObserver from "use-resize-observer";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { Card, CardContent, CardHeader } from "@/components/ui/card";

function convertResourcesToMap(resources: Resource[]): {
  [key: string]: string;
} {
  const resourceMap: { [key: string]: string } = {};
  resources.forEach((resource) => {
    resourceMap[resource.name] = resource.amount.toString();
  });
  return resourceMap;
}

const resourceSchema = z.array(
  z.object({
    name: z.string().min(1, {
      message: "资源名称不能为空",
    }),
    amount: z.number().int().min(0, {
      message: "资源至少为0",
    }),
  }),
);

const formSchema = z.object({
  name: z
    .string()
    .min(1, {
      message: "账户名称不能为空",
    })
    .max(16, {
      message: "账户名称最多16个字符",
    }),
  resources: resourceSchema,
});

type FormSchema = z.infer<typeof formSchema>;
interface ResourceMap {
  [key: string]: string;
}

interface Resource {
  name: string;
  amount: number;
}

type ProjectSheetProps = React.HTMLAttributes<HTMLDivElement> & {
  formData: FormSchema;
  isPending: boolean;
  resourcesData?: Resource[];
  apiProjcet: (form: FormSchema) => void;
};

const ProjectSheet = ({
  formData,
  isPending,
  resourcesData,
  apiProjcet,
  children,
}: ProjectSheetProps) => {
  const { ref: refRoot, width, height } = useResizeObserver();

  const [open, setOpenchange] = useState(false);
  // 1. Define your form.
  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: formData,
  });

  const {
    fields: resourcesFields,
    append: resourcesAppend,
    remove: resourcesRemove,
  } = useFieldArray<FormSchema>({
    name: "resources",
    control: form.control,
  });

  useEffect(() => {
    if (!open) return;
    if (!resourcesData) return;
    form.reset();
    const resourcesList = resourcesData;
    resourcesAppend(resourcesList);
  }, [form, open, resourcesAppend, resourcesData]);

  // 2. Define a submit handler.
  const onSubmit = (values: z.infer<typeof formSchema>) => {
    // Do something with the form values.
    // ✅ This will be type-safe and validated.
    !isPending && apiProjcet(values);
    setOpenchange(false);
  };

  return (
    <Sheet open={open} onOpenChange={setOpenchange}>
      <SheetTrigger asChild>{children}</SheetTrigger>
      {/* scroll in sheet: https://github.com/shadcn-ui/ui/issues/16 */}
      <SheetContent className="flex flex-col gap-6 sm:max-w-3xl">
        <SheetHeader>
          <SheetTitle>新建账户</SheetTitle>
        </SheetHeader>
        <Card
          className="h-[calc(100vh_-104px)] bg-white text-muted-foreground dark:border dark:bg-transparent"
          ref={refRoot}
        >
          <ScrollArea style={{ width, height }}>
            <CardHeader className="py-3"></CardHeader>
            <CardContent>
              <Form {...form}>
                <form
                  // eslint-disable-next-line @typescript-eslint/no-misused-promises
                  onSubmit={form.handleSubmit(onSubmit)}
                  className="grid gap-4 md:grid-cols-1"
                >
                  <FormField
                    control={form.control}
                    name="name"
                    render={({ field }) => (
                      <FormItem className="col-span-1">
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
                  <div className="space-y-2">
                    {resourcesFields.length > 0 && (
                      <div>
                        <FormLabel>资源配额</FormLabel>
                      </div>
                    )}
                    {resourcesFields.map(({ id }, index) => (
                      <div key={id} className="flex items-start gap-2">
                        <FormField
                          control={form.control}
                          name={`resources.${index}.name`}
                          render={({ field }) => (
                            <FormItem className="flex-1">
                              <FormControl>
                                <Input {...field} placeholder="资源名称" />
                              </FormControl>
                              <FormMessage />
                            </FormItem>
                          )}
                        />
                        <FormField
                          control={form.control}
                          name={`resources.${index}.amount`}
                          render={() => (
                            <FormItem>
                              <FormControl>
                                <Input
                                  type="number"
                                  placeholder="资源数目"
                                  {...form.register(
                                    `resources.${index}.amount`,
                                    {
                                      valueAsNumber: true,
                                    },
                                  )}
                                />
                              </FormControl>
                              <FormMessage />
                            </FormItem>
                          )}
                        />
                        <Button
                          size="icon"
                          type="button"
                          variant="outline"
                          onClick={() => resourcesRemove(index)}
                        >
                          <XIcon className="h-4 w-4" />
                        </Button>
                      </div>
                    ))}
                    <Button
                      type="button"
                      variant="secondary"
                      onClick={() => resourcesAppend({ name: "", amount: 0 })}
                    >
                      <CirclePlusIcon className="mr-2 h-4 w-4" />
                      添加配额
                    </Button>
                  </div>
                </form>
              </Form>
            </CardContent>
            <CardContent>
              <LoadableButton
                isLoading={isPending}
                type="submit"
                className="flex flex-col justify-end"
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
            </CardContent>
            <ScrollBar orientation="horizontal" />
          </ScrollArea>
        </Card>
      </SheetContent>
    </Sheet>
  );
};

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
        quota: JSON.stringify(convertResourcesToMap(values.resources)),
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
    },
  });

  const { data: resourcesData } = useQuery({
    queryKey: ["label", "list"],
    queryFn: () => apiResourceList("nvidia.com"),
    select: (res) => {
      return res.data.data.map(
        (item) =>
          ({
            name: item.name,
            amount: 0,
          }) as Resource,
      );
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

  const defaultValues = {
    name: "",
    resources: [
      {
        name: "cpu",
        amount: 0,
      },
      {
        name: "memory",
        amount: 0,
      },
      {
        name: "storage",
        amount: 0,
      },
    ],
  };

  return (
    <div className="flex flex-col gap-2 lg:col-span-3">
      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <div className="flex flex-row justify-between gap-2">
          <div>
            <ProjectSheet
              formData={defaultValues}
              isPending={isPending}
              resourcesData={resourcesData}
              apiProjcet={createNewProject}
            >
              <Button className="h-8">
                <PlusCircleIcon className="-ml-0.5 mr-2 h-4 w-4" />
                新建账户
              </Button>
            </ProjectSheet>
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
  deserved: ResourceMap;
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

  const { mutate: updateProject, isPending } = useMutation({
    mutationFn: (values: z.infer<typeof formSchema>) =>
      apiProjectUpdate({
        name: values.name,
        quota: JSON.stringify(convertResourcesToMap(values.resources)),
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
      toast.success(`账户 ${name} 更新成功`);
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
          const defaultValues = {
            name: proj.name,
            resources: [],
          };
          const resourcesData = Object.entries(proj.deserved).map(
            ([key, value]) =>
              ({
                name: key,
                amount: parseInt(value),
              }) as Resource,
          );
          return (
            <div className="flex flex-col gap-1">
              <Button title="管理用户" onClick={() => navigate(`${proj.id}`)}>
                管理用户
              </Button>
              <ProjectSheet
                formData={defaultValues}
                isPending={isPending}
                resourcesData={resourcesData}
                apiProjcet={updateProject}
              >
                <Button title="修改配额">修改配额</Button>
              </ProjectSheet>
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <Button title="删除账户">删除账户</Button>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>删除账户</AlertDialogTitle>
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
    [deleteProject, isPending, navigate, updateProject],
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
          deserved: JSON.parse(p.deserved) as ResourceMap,
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

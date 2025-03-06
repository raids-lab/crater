import {
  apiAdminAccountList,
  apiAccountCreate,
  apiAccountUpdate,
  IAccount,
} from "@/services/api/account";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { PlusCircleIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { apiProjectDelete } from "@/services/api/account";
import { useAtomValue } from "jotai";
import { globalSettings } from "@/utils/store";
import { DataTable } from "@/components/custom/DataTable";
import { apiResourceList } from "@/services/api/resource";
import { AccountFormSchema, formSchema } from "./account-form";
import { getColumns, toolbarConfig } from "./account-table";
import { Button } from "@/components/ui/button";
import { CalendarIcon, CirclePlusIcon, XIcon } from "lucide-react";
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
import FormLabelMust from "@/components/form/FormLabelMust";
import LoadableButton from "@/components/custom/LoadableButton";
import { z } from "zod";
import { toast } from "sonner";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import { Calendar } from "@/components/ui/calendar";
import { format } from "date-fns";
import { zhCN } from "date-fns/locale";
import { convertFormToQuota, convertQuotaToForm } from "@/utils/quota";
import SelectBox from "@/components/custom/SelectBox";
import { apiAdminUserList } from "@/services/api/admin/user";
import SandwichSheet from "@/components/sheet/SandwichSheet";
import FormImportButton from "@/components/form/FormImportButton";
import FormExportButton from "@/components/form/FormExportButton";
import { MetadataFormAccount } from "@/components/form/types";

export const Account = () => {
  const [isOpen, setIsOpen] = useState(false);
  const { scheduler } = useAtomValue(globalSettings);
  const queryClient = useQueryClient();
  const [cachedFormName, setCachedFormName] = useState<string | null>(null);

  const { data: userList } = useQuery({
    queryKey: ["admin", "userlist"],
    queryFn: apiAdminUserList,
    select: (res) =>
      res.data.data.map((user) => ({
        value: user.id.toString(),
        label: user.attributes.nickname || user.name,
        labelNote: user.name,
      })),
  });

  const query = useQuery({
    queryKey: ["admin", "accounts"],
    queryFn: () => apiAdminAccountList(),
    select: (res) => res.data.data,
  });

  const { mutate: deleteAccount } = useMutation({
    mutationFn: (account: IAccount) => apiProjectDelete(account.id),
    onSuccess: async (_, account) => {
      await queryClient.invalidateQueries({
        queryKey: ["admin", "accounts"],
      });
      toast.success(`账户 ${account.nickname} 已删除`);
    },
  });

  const { mutate: createAccount, isPending: isCreatePending } = useMutation({
    mutationFn: (values: AccountFormSchema) =>
      apiAccountCreate({
        name: values.name,
        quota: convertFormToQuota(values.resources),
        expiredAt: values.expiredAt,
        admins: values.admins?.map((id) => parseInt(id)),
        withoutVolcano: scheduler !== "volcano",
      }),
    onSuccess: async (_, { name }) => {
      await queryClient.invalidateQueries({
        queryKey: ["admin", "accounts"],
      });
      toast.success(`账户 ${name} 新建成功`);
      setIsOpen(false);
    },
  });

  const { mutate: updateAccount, isPending: isUpdatePending } = useMutation({
    mutationFn: (values: AccountFormSchema) =>
      apiAccountUpdate(values.id ?? 0, {
        name: values.name,
        quota: convertFormToQuota(values.resources),
        expiredAt: values.expiredAt,
        admins: values.admins?.map((id) => parseInt(id)),
        withoutVolcano: scheduler !== "volcano",
      }),
    onSuccess: async (_, { name }) => {
      await queryClient.invalidateQueries({
        queryKey: ["admin", "accounts"],
      });
      toast.success(`账户 ${name} 更新成功`);
      setIsOpen(false);
    },
  });

  const { data: resourceDemension } = useQuery({
    queryKey: ["resources", "list"],
    queryFn: () => apiResourceList(false),
    select: (res) => {
      return res.data.data
        .map((item) => item.name)
        .filter(
          (name) =>
            name != "ephemeral-storage" &&
            name != "hugepages-1Gi" &&
            name != "hugepages-2Mi" &&
            name != "pods",
        )
        .sort(
          // cpu > memory > xx/xx > pods
          (a, b) => {
            if (a === "cpu") {
              return -1;
            }
            if (b === "cpu") {
              return 1;
            }
            if (a === "memory") {
              return -1;
            }
            if (b === "memory") {
              return 1;
            }
            return a.localeCompare(b);
          },
        );
    },
  });

  // 1. Define your form.
  const form = useForm<AccountFormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: "",
      resources: [{ name: "cpu" }, { name: "memory" }],
      expiredAt: undefined,
      admins: [],
    },
  });

  const currentValues = form.watch();

  const {
    fields: resourcesFields,
    append: resourcesAppend,
    remove: resourcesRemove,
  } = useFieldArray<AccountFormSchema>({
    name: "resources",
    control: form.control,
  });

  // 2. Define a submit handler.
  const onSubmit = (values: z.infer<typeof formSchema>) => {
    // Do something with the form values.
    // ✅ This will be type-safe and validated.
    if (values.id) {
      updateAccount(values);
    } else {
      createAccount(values);
    }
  };

  const handleCreate = useCallback(() => {
    form.reset({
      name: "",
      resources: resourceDemension?.map((name) => ({ name })) || [],
      expiredAt: undefined,
      admins: [],
    });
    setIsOpen(true);
  }, [form, resourceDemension]);

  const handleEdit = useCallback(
    (account: IAccount) => {
      // open the sheet
      const resources = convertQuotaToForm(account.quota, resourceDemension);
      form.reset({
        id: account.id,
        name: account.nickname,
        resources: resources,
        expiredAt: account.expiredAt ? new Date(account.expiredAt) : undefined,
      });
      setIsOpen(true);
    },
    [resourceDemension, form],
  );

  const columns = useMemo(() => {
    return getColumns(handleEdit, deleteAccount);
  }, [handleEdit, deleteAccount]);

  return (
    <>
      <DataTable
        info={{
          title: "账户管理",
          description:
            "账户可包含多名用户，每个账户可设置资源配额，公共账户默认不设置配额，优先级最低",
        }}
        query={query}
        columns={columns}
        toolbarConfig={toolbarConfig}
      >
        <Button onClick={handleCreate}>
          <PlusCircleIcon className="size-4" />
          新建账户
        </Button>
      </DataTable>
      <SandwichSheet
        isOpen={isOpen}
        onOpenChange={setIsOpen}
        title={form.getValues("id") ? "编辑账户" : "新建账户"}
        description="账户可包含多名用户，每个账户可设置资源配额和过期时间"
        className="sm:max-w-4xl"
        footer={
          <>
            <FormImportButton
              form={form}
              metadata={MetadataFormAccount}
              beforeImport={(data) => {
                setCachedFormName(data.name);
              }}
              afterImport={() => {
                if (cachedFormName) {
                  form.setValue("name", cachedFormName);
                  form.setValue("expiredAt", undefined);
                }
                setCachedFormName(null);
              }}
            />
            <FormExportButton form={form} metadata={MetadataFormAccount} />
            <LoadableButton
              isLoading={isCreatePending || isUpdatePending}
              isLoadingText={form.getValues("id") ? "更新账户" : "新建账户"}
              type="submit"
              onClick={async () => {
                // Trigger validations before submitting
                const isValid = await form.trigger();
                if (isValid) {
                  form.handleSubmit(onSubmit)();
                }
              }}
            >
              <CirclePlusIcon className="size-4" />
              {form.getValues("id") ? "更新账户" : "新建账户"}
            </LoadableButton>
          </>
        }
      >
        <Form {...form}>
          <form
            onSubmit={form.handleSubmit(onSubmit)}
            className="grid gap-4 px-6"
          >
            <div className="flex flex-row items-start justify-between gap-4">
              <FormField
                control={form.control}
                name="name"
                render={({ field }) => (
                  <FormItem className="col-span-1 flex-grow">
                    <FormLabel>
                      账户名称
                      <FormLabelMust />
                    </FormLabel>
                    <FormControl>
                      <Input
                        autoComplete="off"
                        {...field}
                        className="w-full"
                        autoFocus={true}
                      />
                    </FormControl>
                    <FormDescription>
                      账户名称最多16个字符，可以包含汉字，但必须是唯一的
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="expiredAt"
                render={({ field }) => (
                  <FormItem className="col-span-1 flex flex-col space-y-2">
                    <FormLabel className="pb-1 pt-1.5">过期时间</FormLabel>
                    <Popover>
                      <PopoverTrigger asChild>
                        <FormControl>
                          <Button
                            variant={"outline"}
                            className={cn(
                              "w-[240px] pl-3 text-left font-normal",
                              !field.value && "text-muted-foreground",
                            )}
                          >
                            {field.value ? (
                              format(field.value, "PPP", {
                                locale: zhCN,
                              })
                            ) : (
                              <span>请选择日期</span>
                            )}
                            <CalendarIcon className="ml-auto size-4 opacity-50" />
                          </Button>
                        </FormControl>
                      </PopoverTrigger>
                      <PopoverContent className="w-auto p-0" align="start">
                        <Calendar
                          mode="single"
                          locale={zhCN}
                          selected={field.value}
                          onSelect={field.onChange}
                          disabled={(date) => date < new Date()}
                          initialFocus
                        />
                      </PopoverContent>
                    </Popover>
                    <FormDescription>若不填写，账户将永不过期</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
            {!form.getValues("id") && (
              <FormField
                control={form.control}
                name="admins"
                render={() => (
                  <FormItem className="col-span-1 flex-grow">
                    <FormLabel>
                      账户管理员
                      <FormLabelMust />
                    </FormLabel>
                    <FormControl>
                      <SelectBox
                        className="col-span-4 h-8"
                        options={userList ?? []}
                        inputPlaceholder="搜索用户"
                        placeholder="选择用户"
                        value={currentValues.admins}
                        onChange={(value) => form.setValue("admins", value)}
                      />
                    </FormControl>
                    <FormDescription>
                      指定账户管理员后，账户内的用户管理、资源分配等操作，将由账户管理员负责
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}
            <div className="space-y-2">
              {resourcesFields.length > 0 && <FormLabel>资源配额</FormLabel>}
              {resourcesFields.map(({ id }, index) => (
                <div key={id} className="flex flex-row gap-2">
                  <FormField
                    control={form.control}
                    name={`resources.${index}.name`}
                    render={({ field }) => (
                      <FormItem className="w-fit">
                        <FormControl>
                          <Input
                            {...field}
                            placeholder="资源"
                            className="font-mono"
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name={`resources.${index}.guaranteed`}
                    render={() => (
                      <FormItem>
                        <FormControl>
                          <Input
                            type="number"
                            placeholder="保证"
                            className="font-mono"
                            {...form.register(`resources.${index}.guaranteed`, {
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
                    name={`resources.${index}.deserved`}
                    render={() => (
                      <FormItem>
                        <FormControl>
                          <Input
                            type="string"
                            placeholder="应得"
                            className="font-mono"
                            {...form.register(`resources.${index}.deserved`, {
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
                    name={`resources.${index}.capability`}
                    render={() => (
                      <FormItem>
                        <FormControl>
                          <Input
                            type="string"
                            placeholder="上限"
                            className="font-mono"
                            {...form.register(`resources.${index}.capability`, {
                              valueAsNumber: true,
                            })}
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <div>
                    <Button
                      size="icon"
                      type="button"
                      variant="outline"
                      onClick={() => resourcesRemove(index)}
                    >
                      <XIcon className="size-4" />
                    </Button>
                  </div>
                </div>
              ))}
              {resourcesFields.length > 0 && (
                <FormDescription>
                  请输入整数，CPU 资源单位为核数，Memory 资源单位为
                  GB。如果不填写，则保证和应得为 0，上限为无穷大
                </FormDescription>
              )}

              <Button
                type="button"
                variant="secondary"
                onClick={() =>
                  resourcesAppend({
                    name: "",
                  })
                }
              >
                <CirclePlusIcon className="size-4" />
                添加资源配额维度
              </Button>
            </div>
          </form>
        </Form>
      </SandwichSheet>
    </>
  );
};

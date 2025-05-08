// i18n-processed-v1.1.0
import { useTranslation } from "react-i18next";
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
import { getColumns, getToolbarConfig } from "./account-table";
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
import LoadableButton from "@/components/button/LoadableButton";
import { z } from "zod";
import { toast } from "sonner";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import { format } from "date-fns";
import { zhCN } from "date-fns/locale";
import { convertFormToQuota, convertQuotaToForm } from "@/utils/quota";
import SelectBox from "@/components/custom/SelectBox";
import { apiAdminUserList } from "@/services/api/admin/user";
import SandwichSheet, {
  SandwichLayout,
} from "@/components/sheet/SandwichSheet";
import FormImportButton from "@/components/form/FormImportButton";
import FormExportButton from "@/components/form/FormExportButton";
import { MetadataFormAccount } from "@/components/form/types";
import { Calendar } from "@/components/ui/calendar";

export const Account = () => {
  const [isOpen, setIsOpen] = useState(false);
  const { t } = useTranslation();
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
      toast.success(t("toast.accountDeleted", { name: account.nickname }));
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
      toast.success(t("toast.accountCreated", { name }));
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
      toast.success(t("toast.accountUpdated", { name }));
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
    return getColumns(handleEdit, deleteAccount, t);
  }, [handleEdit, deleteAccount, t]);

  return (
    <>
      <DataTable
        info={{
          title: t("accountManagement.title"),
          description: t("accountManagement.description"),
        }}
        storageKey="admin_account_management"
        query={query}
        columns={columns}
        toolbarConfig={getToolbarConfig(t)}
      >
        <Button onClick={handleCreate}>
          <PlusCircleIcon className="size-4" />
          {t("accountForm.createButton")}
        </Button>
      </DataTable>
      <SandwichSheet
        isOpen={isOpen}
        onOpenChange={setIsOpen}
        title={
          form.getValues("id")
            ? t("accountForm.editTitle")
            : t("accountForm.createTitle")
        }
        description={t("accountForm.description")}
        className="sm:max-w-4xl"
      >
        <Form {...form}>
          <form
            onSubmit={form.handleSubmit(onSubmit)}
            className="grid gap-4 px-6"
          >
            <SandwichLayout
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
                  <FormExportButton
                    form={form}
                    metadata={MetadataFormAccount}
                  />
                  <LoadableButton
                    isLoading={isCreatePending || isUpdatePending}
                    isLoadingText={
                      form.getValues("id")
                        ? t("accountForm.updateButton")
                        : t("accountForm.createButton")
                    }
                    type="submit"
                  >
                    <CirclePlusIcon className="size-4" />
                    {form.getValues("id")
                      ? t("accountForm.updateButton")
                      : t("accountForm.createButton")}
                  </LoadableButton>
                </>
              }
            >
              <div className="flex flex-row items-start justify-between gap-4">
                <FormField
                  control={form.control}
                  name="name"
                  render={({ field }) => (
                    <FormItem className="col-span-1 grow">
                      <FormLabel>
                        {t("accountForm.nameLabel")}
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
                        {t("accountForm.nameDescription")}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="expiredAt"
                  render={({ field }) => (
                    <FormItem className="col-span-1 flex flex-col">
                      <FormLabel>{t("accountForm.expiredAtLabel")}</FormLabel>
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
                                <span>
                                  {t("accountForm.expiredAtPlaceholder")}
                                </span>
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
                      <FormDescription>
                        {t("accountForm.expiredAtDescription")}
                      </FormDescription>
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
                    <FormItem>
                      <FormLabel>
                        {t("accountForm.adminsLabel")}
                        <FormLabelMust />
                      </FormLabel>
                      <FormControl>
                        <SelectBox
                          className="my-0 h-8"
                          options={userList ?? []}
                          inputPlaceholder={t(
                            "accountForm.adminsSearchPlaceholder",
                          )}
                          placeholder={t("accountForm.adminsPlaceholder")}
                          value={currentValues.admins}
                          onChange={(value) => form.setValue("admins", value)}
                        />
                      </FormControl>
                      <FormDescription>
                        {t("accountForm.adminsDescription")}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              )}
              <div className="space-y-2">
                {resourcesFields.length > 0 && (
                  <FormLabel>{t("accountForm.quotaLabel")}</FormLabel>
                )}
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
                              placeholder={t("accountForm.resourcePlaceholder")}
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
                              placeholder={t(
                                "accountForm.guaranteedPlaceholder",
                              )}
                              className="font-mono"
                              {...form.register(
                                `resources.${index}.guaranteed`,
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
                    <FormField
                      control={form.control}
                      name={`resources.${index}.deserved`}
                      render={() => (
                        <FormItem>
                          <FormControl>
                            <Input
                              type="string"
                              placeholder={t("accountForm.deservedPlaceholder")}
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
                              placeholder={t(
                                "accountForm.capabilityPlaceholder",
                              )}
                              className="font-mono"
                              {...form.register(
                                `resources.${index}.capability`,
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
                    {t("accountForm.quotaDescription")}
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
                  {t("accountForm.addQuotaButton")}
                </Button>
              </div>
            </SandwichLayout>
          </form>
        </Form>
      </SandwichSheet>
    </>
  );
};

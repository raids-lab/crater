// i18n-processed-v1.1.0
// Modified code
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { FieldPath, FieldValues, UseFormReturn } from "react-hook-form";
import { useQuery } from "@tanstack/react-query";
import {
  apiResourceList,
  apiResourceNetworks,
  Resource,
} from "@/services/api/resource";
import FormLabelMust from "@/components/form/FormLabelMust";
import Combobox, { ComboboxItem } from "@/components/form/Combobox";
import TipBadge from "@/components/badge/TipBadge";
import { ChartNoAxesColumn, CircleHelpIcon } from "lucide-react";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { GrafanaIframe } from "@/pages/Embed/Monitor";
import { useAtomValue } from "jotai";
import { configGrafanaOverviewAtom } from "@/utils/store/config";
import { Switch } from "../ui/switch";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "../ui/tooltip";
import { useMemo } from "react";

interface ResourceFormFieldsProps<T extends FieldValues> {
  form: UseFormReturn<T>;
  cpuPath: FieldPath<T>;
  memoryPath: FieldPath<T>;
  gpuCountPath: FieldPath<T>;
  gpuModelPath: FieldPath<T>;
  rdmaPath?: {
    rdmaEnabled: FieldPath<T>;
    rdmaLabel: FieldPath<T>;
  };
}

export function ResourceFormFields<T extends FieldValues>({
  form,
  cpuPath,
  memoryPath,
  gpuCountPath,
  gpuModelPath,
  rdmaPath,
}: ResourceFormFieldsProps<T>) {
  const { t } = useTranslation();
  const gpuCount = form.watch(gpuCountPath);
  const grafanaOverview = useAtomValue(configGrafanaOverviewAtom);

  // 获取可用资源列表
  const { data: resources } = useQuery({
    queryKey: ["resources", "list"],
    queryFn: () => apiResourceList(true),
    select: (res) => {
      return res.data.data
        .sort((a, b) => {
          return b.amountSingleMax - a.amountSingleMax;
        })
        .filter((item) => item.amountSingleMax > 0)
        .map(
          (item) =>
            ({
              value: item.name,
              label: item.label,
              detail: item,
            }) as ComboboxItem<Resource>,
        );
    },
  });

  return (
    <>
      <div className="grid grid-cols-3 items-start gap-3">
        <FormField
          control={form.control}
          name={cpuPath}
          render={() => (
            <FormItem>
              <FormLabel>
                {t("resourceForm.cpuLabel")}
                <FormLabelMust />
              </FormLabel>
              <FormControl>
                <Input
                  type="number"
                  {...form.register(cpuPath, { valueAsNumber: true })}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name={memoryPath}
          render={() => (
            <FormItem>
              <FormLabel>
                {t("resourceForm.memoryLabel")}
                <FormLabelMust />
              </FormLabel>
              <FormControl>
                <Input
                  type="number"
                  {...form.register(memoryPath, { valueAsNumber: true })}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name={gpuCountPath}
          render={() => (
            <FormItem>
              <FormLabel>
                {t("resourceForm.gpuCountLabel")}
                <FormLabelMust />
              </FormLabel>
              <FormControl>
                <Input
                  type="number"
                  {...form.register(gpuCountPath, { valueAsNumber: true })}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
      </div>
      <FormField
        control={form.control}
        name={gpuModelPath}
        render={({ field }) => (
          <FormItem hidden={gpuCount === 0}>
            <FormLabel>
              {t("resourceForm.gpuModelLabel")}
              <FormLabelMust />
            </FormLabel>
            <FormControl>
              <Combobox
                items={resources ?? []}
                renderLabel={(item) => (
                  <div className="flex w-full flex-row items-center justify-between gap-3">
                    <p>{item.label}</p>
                    <TipBadge
                      title={t("resourceForm.gpuTip", {
                        max: item.detail?.amountSingleMax,
                      })}
                      className="bg-highlight-purple/15 text-highlight-purple"
                    />
                  </div>
                )}
                current={field.value ?? ""}
                handleSelect={(value) => {
                  field.onChange(value);
                  if (rdmaPath) {
                    form.resetField(rdmaPath.rdmaEnabled);
                    form.resetField(rdmaPath.rdmaLabel);
                  }
                }}
                formTitle={t("resourceForm.gpuComboboxTitle")}
              />
            </FormControl>
            <FormMessage />
          </FormItem>
        )}
      />
      {rdmaPath && (
        <RDMAFormFields
          form={form}
          resources={resources ?? []}
          gpuModelPath={gpuModelPath}
          rdmaPath={rdmaPath}
        />
      )}
      <div>
        <Sheet>
          <SheetTrigger asChild>
            <Button
              type="button"
              variant="secondary"
              className="cursor-pointer"
            >
              <ChartNoAxesColumn className="size-4" />
              {t("resourceForm.freeResourceButton")}
            </Button>
          </SheetTrigger>
          <SheetContent className="sm:max-w-4xl">
            <SheetHeader>
              <SheetTitle>
                {t("resourceForm.freeResourceSheetTitle")}
              </SheetTitle>
            </SheetHeader>
            <div className="h-[calc(100vh-6rem)] w-full px-4">
              <GrafanaIframe baseSrc={grafanaOverview.schedule} />
            </div>
          </SheetContent>
        </Sheet>
      </div>
    </>
  );
}

function RDMAFormFields<T extends FieldValues>({
  form,
  resources,
  gpuModelPath,
  rdmaPath,
}: {
  form: UseFormReturn<T>;
  resources: ComboboxItem<Resource>[];
  gpuModelPath: FieldPath<T>;
  rdmaPath: {
    rdmaEnabled: FieldPath<T>;
    rdmaLabel: FieldPath<T>;
  };
}) {
  const { t } = useTranslation();
  const gpuModel = form.watch(gpuModelPath);
  const rdmaEnabled = form.watch(rdmaPath.rdmaEnabled);
  const gpuID = useMemo(() => {
    if (gpuModel) {
      const gpu = resources?.find((item) => item.value === gpuModel);
      return gpu?.detail?.ID ?? 0;
    }
    return 0;
  }, [gpuModel, resources]);

  // 获取给定的 GPU 型号对应的网络资源列表
  const { data: networks } = useQuery({
    queryKey: ["resources", "networks", "list", gpuID],
    queryFn: () => apiResourceNetworks(gpuID),
    select: (res) =>
      res.data.data
        .filter((item) => item.amountSingleMax > 0)
        .map(
          (item) =>
            ({
              value: item.name,
              label: item.label,
              detail: item,
            }) as ComboboxItem<Resource>,
        ),
  });

  return (
    <>
      {networks && networks.length > 0 && (
        <div className="flex flex-col gap-1.5">
          <FormField
            control={form.control}
            name={rdmaPath.rdmaEnabled}
            render={({ field }) => (
              <FormItem>
                <div className="flex flex-row items-center justify-between space-y-0 space-x-0">
                  <FormLabel>
                    {t("resourceForm.rdmaLabel")}
                    <TooltipProvider delayDuration={100}>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <CircleHelpIcon className="text-muted-foreground size-4 hover:cursor-help" />
                        </TooltipTrigger>
                        <TooltipContent>
                          <h2 className="mb-0.5 font-semibold">
                            {t("resourceForm.tooltip.title")}
                          </h2>
                          <p>{t("resourceForm.tooltip.line1")}</p>
                          <p>{t("resourceForm.tooltip.line2")}</p>
                          <p>{t("resourceForm.tooltip.line3")}</p>
                          <p>{t("resourceForm.tooltip.line4")}</p>
                          <p>{t("resourceForm.tooltip.line5")}</p>
                        </TooltipContent>
                      </Tooltip>
                    </TooltipProvider>
                  </FormLabel>
                  <FormControl>
                    <Switch
                      checked={field.value}
                      onCheckedChange={(value) => field.onChange(value)}
                    />
                  </FormControl>
                </div>
                <FormMessage />
              </FormItem>
            )}
          />
          {rdmaEnabled && (
            <FormField
              control={form.control}
              name={rdmaPath.rdmaLabel}
              render={({ field }) => (
                <FormItem>
                  <FormControl>
                    <Combobox
                      items={networks}
                      renderLabel={(item) => (
                        <div className="flex w-full flex-row items-center justify-between gap-3">
                          <p>{item.label}</p>
                        </div>
                      )}
                      current={field.value ?? ""}
                      handleSelect={(value) => {
                        field.onChange(value);
                      }}
                      formTitle={t("resourceForm.rdmaNetworkTitle")}
                    />
                  </FormControl>
                  <FormDescription>
                    {t("resourceForm.rdmaDescription", {
                      field: "RDMA",
                    })}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          )}
        </div>
      )}
    </>
  );
}

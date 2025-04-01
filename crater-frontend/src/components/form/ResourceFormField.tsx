import { Button } from "@/components/ui/button";
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { FieldPath, FieldValues, UseFormReturn } from "react-hook-form";
import { useQuery } from "@tanstack/react-query";
import { apiResourceList } from "@/services/api/resource";
import FormLabelMust from "@/components/form/FormLabelMust";
import Combobox from "@/components/form/Combobox";
import TipBadge from "@/components/badge/TipBadge";
import { ChartNoAxesColumn } from "lucide-react";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { GrafanaIframe } from "@/pages/Embed/Monitor";
import { useAtomValue } from "jotai";
import { asyncGrafanaOverviewAtom } from "@/utils/store/config";

interface ResourceFormFieldsProps<T extends FieldValues> {
  form: UseFormReturn<T>;
  cpuPath: FieldPath<T>;
  memoryPath: FieldPath<T>;
  gpuCountPath: FieldPath<T>;
  gpuModelPath: FieldPath<T>;
  required?: boolean;
}

export function ResourceFormFields<T extends FieldValues>({
  form,
  cpuPath,
  memoryPath,
  gpuCountPath,
  gpuModelPath,
  required = true,
}: ResourceFormFieldsProps<T>) {
  const gpuCount = form.watch(gpuCountPath);
  const grafanaOverview = useAtomValue(asyncGrafanaOverviewAtom);

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
        .map((item) => ({
          value: item.name,
          label: item.label.toUpperCase(),
          detail: item,
        }));
    },
  });

  return (
    <>
      <div className="grid grid-cols-3 gap-3">
        <FormField
          control={form.control}
          name={cpuPath}
          render={() => (
            <FormItem>
              <FormLabel>
                CPU (核数)
                {required && <FormLabelMust />}
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
                内存 (GiB)
                {required && <FormLabelMust />}
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
                GPU (卡数)
                {required && <FormLabelMust />}
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
              GPU 型号
              {required && <FormLabelMust />}
            </FormLabel>
            <FormControl>
              <Combobox
                items={resources ?? []}
                renderLabel={(item) => (
                  <div className="flex w-full flex-row items-center justify-between gap-3">
                    <p>{item.label}</p>
                    <TipBadge
                      title={`可申请至多 ${item.detail?.amountSingleMax} 张卡`}
                      className="bg-highlight-purple/15 text-highlight-purple"
                    />
                  </div>
                )}
                current={field.value ?? ""}
                handleSelect={(value) => field.onChange(value)}
                formTitle="GPU 型号"
              />
            </FormControl>
            <FormMessage />
          </FormItem>
        )}
      />
      <div>
        <Sheet>
          <SheetTrigger asChild>
            <Button
              type="button"
              variant="secondary"
              className="cursor-pointer"
            >
              <ChartNoAxesColumn className="size-4" />
              空闲资源查询
            </Button>
          </SheetTrigger>
          <SheetContent className="sm:max-w-4xl">
            <SheetHeader>
              <SheetTitle>空闲资源查询</SheetTitle>
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

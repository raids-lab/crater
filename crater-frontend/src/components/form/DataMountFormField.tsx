// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-nocheck
import { useState } from "react";
import { ArrayPath, Path, UseFormReturn, useFieldArray } from "react-hook-form";
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
import { Separator } from "@/components/ui/separator";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";
import { CirclePlus, XIcon } from "lucide-react";
import FormLabelMust from "@/components/form/FormLabelMust";
import { FileSelectDialog } from "@/components/file/FileSelectDialog";
import Combobox from "@/components/form/Combobox";
import DatasetItem from "@/components/form/DatasetItem";
import AccordionCard from "@/components/form/AccordionCard";
import { useQuery } from "@tanstack/react-query";
import { apiGetDataset, Dataset } from "@/services/api/dataset";
import { useAtomValue } from "jotai";
import { globalUserInfo } from "@/utils/store";
import { ComboboxItem } from "@/components/form/Combobox";
import { VolumeMountsSchema, VolumeMountType } from "@/utils/form";

const DataMountCard = "数据挂载";

interface VolumeMountsCardProps<
  T extends {
    volumeMounts: VolumeMountsSchema;
  },
> {
  form: UseFormReturn<T>;
  className?: string;
}

export function VolumeMountsCard<
  T extends {
    volumeMounts: VolumeMountsSchema;
  },
>({ form, className }: VolumeMountsCardProps<T>) {
  const [dataMountOpen, setDataMountOpen] = useState<string>(DataMountCard);
  const user = useAtomValue(globalUserInfo);

  // Get dataset information
  const datasetInfo = useQuery({
    queryKey: ["datsets"],
    queryFn: () => apiGetDataset(),
    select: (res) => {
      return res.data.data.map(
        (item) =>
          ({
            value: item.id.toString(),
            label: item.name,
            detail: item,
          }) as ComboboxItem<Dataset>,
      );
    },
  });

  // Field array for volume mounts
  const {
    fields: volumeMountFields,
    append: volumeMountAppend,
    remove: volumeMountRemove,
  } = useFieldArray({
    name: "volumeMounts" as ArrayPath<T>,
    control: form.control,
  });

  const resetVolumeMountsFields = (index: number, type: number) => {
    form.setValue(`volumeMounts.${index}` as Path<T>, {
      type: type,
      subPath: "",
      mountPath: "",
    });
  };

  return (
    <AccordionCard
      cardTitle={DataMountCard}
      value={dataMountOpen}
      setValue={setDataMountOpen}
      className={className}
    >
      <div className="mt-3 space-y-5">
        {volumeMountFields.map((field, index) => (
          <div key={field.id}>
            <Separator className={cn("mb-5", index === 0 && "hidden")} />
            <div className="space-y-5">
              <FormField
                control={form.control}
                name={`volumeMounts.${index}.subPath`}
                render={({ field }) => {
                  const disabled =
                    form.getValues(`volumeMounts.${index}.mountPath`) ===
                    `/home/${user.name}`;
                  return (
                    <FormItem className="relative">
                      <FormLabel>
                        挂载源 {index + 1}
                        <FormLabelMust />
                      </FormLabel>
                      <button
                        type="button"
                        onClick={() => {
                          volumeMountRemove(index);
                        }}
                        className="data-[state=open]:bg-accent data-[state=open]:text-muted-foreground absolute -top-1.5 right-0 rounded-sm opacity-50 transition-opacity hover:opacity-100 focus:outline-hidden disabled:pointer-events-none"
                      >
                        <XIcon className="size-4" />
                        <span className="sr-only">Close</span>
                      </button>
                      <FormControl>
                        <Tabs
                          value={
                            form.getValues(`volumeMounts.${index}.type`) ===
                            VolumeMountType.FileType
                              ? "file"
                              : "dataset"
                          }
                          onValueChange={(value) => {
                            form.setValue(
                              `volumeMounts.${index}.type`,
                              value === "file"
                                ? VolumeMountType.FileType
                                : VolumeMountType.DataType,
                            );
                          }}
                          className="w-full"
                        >
                          <TabsList className="grid w-full grid-cols-2">
                            <TabsTrigger
                              value="file"
                              onClick={() =>
                                resetVolumeMountsFields(
                                  index,
                                  VolumeMountType.FileType,
                                )
                              }
                              disabled={disabled}
                            >
                              文件
                            </TabsTrigger>
                            <TabsTrigger
                              value="dataset"
                              onClick={() =>
                                resetVolumeMountsFields(
                                  index,
                                  VolumeMountType.DataType,
                                )
                              }
                              disabled={disabled}
                            >
                              数据集
                            </TabsTrigger>
                          </TabsList>
                          <TabsContent value="file">
                            <FileSelectDialog
                              value={field.value.split("/").pop()}
                              handleSubmit={(item) => {
                                field.onChange(item.id);
                                form.setValue(
                                  `volumeMounts.${index}.type`,
                                  VolumeMountType.FileType,
                                );
                                form.setValue(
                                  `volumeMounts.${index}.mountPath`,
                                  `/data/${item.name}`,
                                );
                              }}
                              disabled={disabled}
                            />
                          </TabsContent>
                          <TabsContent value="dataset">
                            <Combobox
                              items={datasetInfo.data ?? []}
                              current={field.value}
                              disabled={disabled}
                              renderLabel={(item) => (
                                <DatasetItem item={item} />
                              )}
                              handleSelect={(value) => {
                                field.onChange(value);
                                form.setValue(
                                  `volumeMounts.${index}.type`,
                                  VolumeMountType.DataType,
                                );
                                form.setValue(
                                  `volumeMounts.${index}.datasetID`,
                                  Number(value),
                                );
                                form.setValue(
                                  `volumeMounts.${index}.mountPath`,
                                  `/data/${datasetInfo.data?.find((item) => item.value === value)?.detail?.name}`,
                                );
                              }}
                              formTitle="数据集"
                            />
                          </TabsContent>
                        </Tabs>
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  );
                }}
              />
              <FormField
                control={form.control}
                name={`volumeMounts.${index}.mountPath`}
                render={({ field }) => {
                  const disabled =
                    form.getValues(`volumeMounts.${index}.mountPath`) ===
                    `/home/${user.name}`;
                  return (
                    <FormItem>
                      <FormLabel>
                        挂载点 {index + 1}
                        <FormLabelMust />
                      </FormLabel>
                      <FormControl>
                        <Input {...field} className="font-mono" />
                      </FormControl>
                      <FormDescription>
                        {disabled
                          ? "默认持久化用户主目录，请谨慎修改"
                          : "可修改容器中的挂载路径"}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  );
                }}
              />
            </div>
          </div>
        ))}
        <Button
          type="button"
          variant="secondary"
          className="w-full cursor-pointer"
          onClick={() =>
            volumeMountAppend({
              type: VolumeMountType.FileType,
              subPath: "",
              mountPath: "",
            })
          }
        >
          <CirclePlus className="size-4" />
          添加{DataMountCard}
        </Button>
      </div>
    </AccordionCard>
  );
}

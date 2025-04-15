// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-nocheck
import { useState } from "react";
import { ArrayPath, UseFormReturn, useFieldArray } from "react-hook-form";
import { Button } from "@/components/ui/button";
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import { CirclePlus, XIcon } from "lucide-react";
import FormLabelMust from "@/components/form/FormLabelMust";
import AccordionCard from "@/components/form/AccordionCard";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ForwardType } from "@/utils/form";

export const ForwardCard = "转发规则";

interface ForwardFormCardProps<
  T extends {
    forwards: Array<{ type: number; name: string; port: number }>;
  },
> {
  form: UseFormReturn<T>;
  className?: string;
}

export function ForwardFormCard<
  T extends {
    forwards: Array<{ type: number; name: string; port: number }>;
  },
>({ form, className }: ForwardFormCardProps<T>) {
  const [forwardOpen, setForwardOpen] = useState<boolean>(true);

  // Field array for forwards
  const {
    fields: forwardFields,
    append: forwardAppend,
    remove: forwardRemove,
  } = useFieldArray({
    name: "forwards" as ArrayPath<T>,
    control: form.control,
  });

  const resetForwardFields = (index: number, type: number) => {
    form.setValue(`forwards.${index}` as ArrayPath<T>, {
      type: type,
      name: "",
      port: null,
    });
  };

  return (
    <AccordionCard
      cardTitle={ForwardCard}
      open={forwardOpen}
      setOpen={setForwardOpen}
      className={className}
    >
      <div className="mt-3 space-y-5">
        {forwardFields.map((field, index) => (
          <div key={field.id} className="relative">
            <Separator className={index === 0 ? "hidden" : "mb-5"} />
            <div className="space-y-5">
              <FormField
                control={form.control}
                name={`forwards.${index}.type`}
                render={() => (
                  <FormItem>
                    <FormLabel>
                      转发类型 {index + 1}
                      <FormLabelMust />
                    </FormLabel>
                    <FormControl>
                      <Tabs
                        value={
                          form.getValues(`forwards.${index}.type`) ===
                          ForwardType.IngressType
                            ? "ingress"
                            : "nodeport"
                        }
                        onValueChange={(value) => {
                          if (index === 0) return; // Prevent modification for the first entry
                          const newType =
                            value === "ingress"
                              ? ForwardType.IngressType
                              : ForwardType.NodeportType;
                          form.setValue(`forwards.${index}.type`, newType);
                          resetForwardFields(index, newType);
                        }}
                        className="w-full"
                      >
                        <TabsList className="grid w-full grid-cols-2">
                          <TabsTrigger value="ingress">
                            Ingress 规则
                          </TabsTrigger>
                          <TabsTrigger value="nodeport">
                            NodePort 规则
                          </TabsTrigger>
                        </TabsList>
                      </Tabs>
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name={`forwards.${index}.name`}
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      规则名称 {index + 1}
                      <FormLabelMust />
                    </FormLabel>
                    <FormControl>
                      <Input {...field} disabled={index === 0} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name={`forwards.${index}.port`}
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      端口号 {index + 1}
                      <FormLabelMust />
                    </FormLabel>
                    <FormControl>
                      <Input
                        {...field}
                        type="number"
                        disabled={index === 0}
                        onChange={(e) => {
                          if (index === 0) return; // Prevent modification for the first entry
                          const value = e.target.value;
                          if (value === "") {
                            field.onChange(null);
                          } else {
                            const parsed = parseInt(value, 10);
                            if (!isNaN(parsed)) {
                              field.onChange(parsed);
                            }
                          }
                        }}
                        value={field.value ?? ""}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              {index > 0 && ( // Add remove button for entries except the first one
                <button
                  type="button"
                  onClick={() => forwardRemove(index)}
                  className="absolute top-0 right-0 mt-1 mr-1 p-1 text-gray-500 hover:text-gray-700 focus:outline-none"
                >
                  <XIcon className="h-4 w-4" />
                  <span className="sr-only">Remove</span>
                </button>
              )}
            </div>
          </div>
        ))}
        <Button
          type="button"
          variant="secondary"
          className="w-full"
          onClick={() =>
            forwardAppend({
              type: ForwardType.IngressType,
              name: "",
              port: null, // Set port to null for no default value
            })
          }
        >
          <CirclePlus className="size-4" />
          添加{ForwardCard}
        </Button>
      </div>
    </AccordionCard>
  );
}

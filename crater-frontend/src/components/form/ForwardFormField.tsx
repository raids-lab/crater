// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-nocheck
import { Fragment, useState } from "react";
import { ArrayPath, UseFormReturn, useFieldArray } from "react-hook-form";
import { Button } from "@/components/ui/button";
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import { CirclePlus, XIcon } from "lucide-react";
import FormLabelMust from "@/components/form/FormLabelMust";
import AccordionCard from "@/components/form/AccordionCard";
import { cn } from "@/lib/utils";

export const ForwardCard = "转发规则";

interface ForwardFormCardProps<
  T extends {
    forwards: Array<{ name: string; port: number }>;
  },
> {
  form: UseFormReturn<T>;
  className?: string;
}

export function ForwardFormCard<
  T extends {
    forwards: Array<{ name: string; port: number }>;
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

  return (
    <AccordionCard
      cardTitle={ForwardCard}
      open={forwardOpen}
      setOpen={setForwardOpen}
      className={className}
    >
      <div className="mt-3 space-y-5">
        {forwardFields.map((field, index) => (
          <Fragment key={field.id}>
            <Separator className={cn("mb-5", index === 0 && "hidden")} />
            <div key={field.id} className="relative">
              <div className="space-y-5">
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
                        <Input {...field} />
                      </FormControl>
                      <FormDescription>
                        仅允许包含小写字母，最多20字符
                      </FormDescription>
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
                          onChange={(e) => {
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
                <button
                  type="button"
                  onClick={() => forwardRemove(index)}
                  className="hover:text-secondary-foreground/50 text-muted-foreground absolute top-0 right-0 mt-1 mr-1 -translate-y-3 cursor-pointer p-1 focus:outline-none"
                >
                  <XIcon className="h-4 w-4" />
                  <span className="sr-only">Remove</span>
                </button>
              </div>
            </div>
          </Fragment>
        ))}
        <Button
          type="button"
          variant="secondary"
          className="w-full"
          onClick={() =>
            forwardAppend({
              name: "",
              port: null, // Set port to null for no default value
            })
          }
        >
          <CirclePlus className="size-4" />
          添加端口转发规则
        </Button>
      </div>
    </AccordionCard>
  );
}

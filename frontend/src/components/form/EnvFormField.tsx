// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-nocheck
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
import { cn } from "@/lib/utils";
import { CirclePlus, XIcon } from "lucide-react";
import FormLabelMust from "@/components/form/FormLabelMust";
import AccordionCard from "@/components/form/AccordionCard";

export const EnvCard = "环境变量";

interface EnvFormCardProps<
  T extends {
    envs: Array<{ name: string; value: string }>;
  },
> {
  form: UseFormReturn<T>;
  open?: boolean;
  setOpen?: (open: boolean) => void;
  className?: string;
  cardTitle?: string;
}

export function EnvFormCard<
  T extends {
    envs: Array<{ name: string; value: string }>;
  },
>({
  form,
  open,
  setOpen,
  className,
  cardTitle = EnvCard,
}: EnvFormCardProps<T>) {
  // Field array for environment variables
  const {
    fields: envFields,
    append: envAppend,
    remove: envRemove,
  } = useFieldArray({
    name: "envs" as ArrayPath<T>,
    control: form.control,
  });

  return (
    <AccordionCard
      cardTitle={cardTitle}
      open={open}
      setOpen={setOpen}
      className={className}
    >
      <div className="mt-3 space-y-5">
        {envFields.map((field, index) => (
          <div key={field.id}>
            <Separator className={cn("mb-5", index === 0 && "hidden")} />
            <div className="space-y-5">
              <FormField
                control={form.control}
                name={`envs.${index}.name`}
                render={({ field }) => (
                  <FormItem className="relative">
                    <button
                      onClick={() => envRemove(index)}
                      className="data-[state=open]:bg-accent data-[state=open]:text-muted-foreground absolute -top-1.5 right-0 rounded-sm opacity-50 transition-opacity hover:opacity-100 focus:outline-hidden disabled:pointer-events-none"
                    >
                      <XIcon className="size-4" />
                      <span className="sr-only">Close</span>
                    </button>
                    <FormLabel>
                      变量名 {index + 1}
                      <FormLabelMust />
                    </FormLabel>
                    <FormControl>
                      <Input {...field} className="font-mono" />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name={`envs.${index}.value`}
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      变量值 {index + 1}
                      <FormLabelMust />
                    </FormLabel>
                    <FormControl>
                      <Input {...field} className="font-mono" />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
          </div>
        ))}
        <Button
          type="button"
          variant="secondary"
          className="w-full"
          onClick={() =>
            envAppend({
              name: "",
              value: "",
            })
          }
        >
          <CirclePlus className="size-4" />
          添加{cardTitle}
        </Button>
      </div>
    </AccordionCard>
  );
}

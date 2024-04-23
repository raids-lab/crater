import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  Form,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { useForm } from "react-hook-form";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  apiNodeLabelsCreate,
  apiNodeLabelsUpdate,
} from "@/services/api/nodelabel";
import * as React from "react";
import { Check, ChevronsUpDown } from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
} from "@/components/ui/command";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { logger } from "@/utils/loglevel";
import { LabelType, getTypeText } from "@/pages/Admin/Cluster/Label/index.tsx";

interface TaskFormProps extends React.HTMLAttributes<HTMLDivElement> {
  closeSheet: () => void;
}

interface UpdateTaskFormProps extends React.HTMLAttributes<HTMLDivElement> {
  closeSheet: () => void;
  labelId: number;
  nowLabelName: string;
  nowLabelPriority: number;
  nowLabelType: LabelType;
}

const typeOptions = [
  {
    label: "GPU",
    value: "1",
  },
];

const formSchema = z.object({
  name: z.string(),
  priority: z.number().int().positive(),
  type: z.string(),
});

const updateFormSchema = formSchema.extend({
  id: z.number().int().positive(),
  name: z.string(),
  priority: z.number().int().positive(),
  type: z.string(),
});

type FormSchema = z.infer<typeof formSchema>;
type UpdateFormSchema = z.infer<typeof updateFormSchema>;

export function NewLabelForm({ closeSheet }: TaskFormProps) {
  const queryClient = useQueryClient();

  const [open, setOpen] = React.useState(false);
  const [value, setValue] = React.useState("");

  const { mutate: createLabelPack } = useMutation({
    mutationFn: (values: FormSchema) =>
      apiNodeLabelsCreate(values.name, values.priority, values.type),
    onSuccess: async (_, { name }) => {
      await queryClient.invalidateQueries({
        queryKey: ["label", "list"],
      });
      toast.success(`Label ${name} 创建成功`);
      closeSheet();
    },
  });

  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: "",
      priority: 0,
      type: "1",
    },
  });

  // 2. Define a submit handler.
  const onSubmit = (values: FormSchema) => {
    createLabelPack(values);
  };

  return (
    <Form {...form}>
      <form
        // eslint-disable-next-line @typescript-eslint/no-misused-promises
        onSubmit={form.handleSubmit(onSubmit)}
        className="mt-6 flex flex-col space-y-4"
      >
        <div className="grid grid-cols-3 gap-3">
          <div className="col-span-3">
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    名称<span className="ml-1 text-red-500">*</span>
                  </FormLabel>
                  <Input {...field} />
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="priority"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    优先级<span className="ml-1 text-red-500">*</span>
                  </FormLabel>
                  <Input
                    {...field}
                    type="number"
                    onChange={(e) => {
                      const value = parseInt(e.target.value, 10);
                      field.onChange(value);
                    }}
                  />
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="type"
              render={() => (
                <FormItem>
                  <FormLabel>
                    类型<span className="ml-1 text-red-500">*</span>
                  </FormLabel>
                  <Popover open={open} onOpenChange={setOpen}>
                    <PopoverTrigger asChild>
                      <Button
                        variant="outline"
                        role="combobox"
                        aria-expanded={open}
                        className="w-[200px] justify-between"
                      >
                        {value
                          ? typeOptions.find(
                              (typeOptions) => typeOptions.value === value,
                            )?.label
                          : "Select framework..."}
                        <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
                      </Button>
                    </PopoverTrigger>
                    <PopoverContent className="w-[200px] p-0">
                      <Command>
                        <CommandInput placeholder="Search framework..." />
                        <CommandEmpty>No framework found.</CommandEmpty>
                        <CommandGroup>
                          {typeOptions.map((framework) => (
                            <CommandItem
                              key={framework.value}
                              value={framework.value}
                              onSelect={(currentValue) => {
                                setValue(
                                  currentValue === value ? "" : currentValue,
                                );
                                setOpen(false);
                              }}
                            >
                              <Check
                                className={cn(
                                  "mr-2 h-4 w-4",
                                  value === framework.value
                                    ? "opacity-100"
                                    : "opacity-0",
                                )}
                              />
                              {framework.label}
                            </CommandItem>
                          ))}
                        </CommandGroup>
                      </Command>
                    </PopoverContent>
                  </Popover>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
        </div>
        <div className="mt-4 flex justify-end">
          <Button type="submit">创建</Button>
        </div>
      </form>
    </Form>
  );
}

export function UpdateLabelForm({
  closeSheet,
  labelId,
  nowLabelName,
  nowLabelPriority,
  nowLabelType,
}: UpdateTaskFormProps) {
  const queryClient = useQueryClient();

  const form = useForm<UpdateFormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: nowLabelName,
      priority: nowLabelPriority,
      type: getTypeText(nowLabelType),
    },
  });

  const closeDialog = (event: { preventDefault: () => void }) => {
    event.preventDefault();
    closeSheet();
  };

  const { mutate: updateLabelPack } = useMutation({
    mutationFn: (values: UpdateFormSchema) =>
      apiNodeLabelsUpdate(labelId, values.name, values.priority, values.type),
    onSuccess: async (_, { name }) => {
      await queryClient.invalidateQueries({
        queryKey: ["label", "list"],
      });
      toast.success(`Label ${name} 更新成功`);
      closeSheet();
    },
  });

  // 2. Define a submit handler.
  const onUpdateSubmit = (values: UpdateFormSchema) => {
    logger.info("values", values, labelId);
    updateLabelPack(values);
  };

  const [open, setOpen] = React.useState(false);
  const [value, setValue] = React.useState("");

  return (
    <Form {...form}>
      <form
        // eslint-disable-next-line @typescript-eslint/no-misused-promises
        onSubmit={form.handleSubmit(onUpdateSubmit)}
        className="mt-6 flex flex-col space-y-4"
      >
        <div className="grid grid-cols-3 gap-3">
          <div className="col-span-3">
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    名称<span className="ml-1 text-red-500">*</span>
                  </FormLabel>
                  <Input {...field} />
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="priority"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    优先级<span className="ml-1 text-red-500">*</span>
                  </FormLabel>
                  <Input
                    {...field}
                    type="number"
                    onChange={(e) => {
                      const value = parseInt(e.target.value, 10);
                      field.onChange(value);
                    }}
                  />
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="type"
              render={() => (
                <FormItem>
                  <FormLabel>
                    类型<span className="ml-1 text-red-500">*</span>
                  </FormLabel>
                  <Popover open={open} onOpenChange={setOpen}>
                    <PopoverTrigger asChild>
                      <Button
                        variant="outline"
                        role="combobox"
                        aria-expanded={open}
                        className="w-[200px] justify-between"
                      >
                        {value
                          ? typeOptions.find(
                              (typeOptions) => typeOptions.value === value,
                            )?.label
                          : "Select framework..."}
                        <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
                      </Button>
                    </PopoverTrigger>
                    <PopoverContent className="w-[200px] p-0">
                      <Command>
                        <CommandInput placeholder="Search framework..." />
                        <CommandEmpty>No framework found.</CommandEmpty>
                        <CommandGroup>
                          {typeOptions.map((framework) => (
                            <CommandItem
                              key={framework.value}
                              value={framework.value}
                              onSelect={(currentValue) => {
                                setValue(
                                  currentValue === value ? "" : currentValue,
                                );
                                setOpen(false);
                              }}
                            >
                              <Check
                                className={cn(
                                  "mr-2 h-4 w-4",
                                  value === framework.value
                                    ? "opacity-100"
                                    : "opacity-0",
                                )}
                              />
                              {framework.label}
                            </CommandItem>
                          ))}
                        </CommandGroup>
                      </Command>
                    </PopoverContent>
                  </Popover>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
          <div className="mt-4 flex justify-between">
            <Button type="submit">提交</Button>
            <Button onClick={closeDialog}>取消</Button>
          </div>
        </div>
      </form>
    </Form>
  );
}

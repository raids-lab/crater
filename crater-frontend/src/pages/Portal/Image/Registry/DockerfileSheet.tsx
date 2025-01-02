import { useForm, UseFormReturn } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { z } from "zod";
import { toast } from "sonner";
import SandwichSheet, {
  SandwichSheetProps,
} from "@/components/sheet/SandwichSheet";
import LoadableButton from "@/components/custom/LoadableButton";
import { PackagePlusIcon } from "lucide-react";
import FormImportButton from "@/components/form/FormImportButton";
import FormExportButton from "@/components/form/FormExportButton";
import { MetadataFormDockerfile } from "@/components/form/types";
import { Input } from "@/components/ui/input";
import { apiUserCreateKaniko } from "@/services/api/imagepack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { DockerfileEditor } from "./DockerfileEditor";
import FormLabelMust from "@/components/form/FormLabelMust";

export const dockerfileFormSchema = z.object({
  dockerfile: z.string().min(1, "Dockerfile content is required"),
  description: z.string().optional(),
});

export type DockerfileFormValues = z.infer<typeof dockerfileFormSchema>;

interface DockerfileSheetContentProps {
  form: UseFormReturn<DockerfileFormValues>;
  onSubmit: (values: DockerfileFormValues) => void;
}

function DockerfileSheetContent({
  form,
  onSubmit,
}: DockerfileSheetContentProps) {
  return (
    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4 px-6">
      <FormField
        control={form.control}
        name="dockerfile"
        render={({ field }) => (
          <FormItem>
            <FormLabel>
              Dockerfile
              <FormLabelMust />
            </FormLabel>
            <FormControl>
              <DockerfileEditor value={field.value} onChange={field.onChange} />
            </FormControl>
            <FormMessage />
          </FormItem>
        )}
      />
      <FormField
        control={form.control}
        name="description"
        render={({ field }) => (
          <FormItem>
            <FormLabel>备注</FormLabel>
            <FormControl>
              <Input {...field} />
            </FormControl>
            <FormDescription>
              关于此镜像的简短描述，如包含的软件版本、用途等。
            </FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />
    </form>
  );
}

interface DockerfileSheetProps extends SandwichSheetProps {
  closeSheet: () => void;
}

export function DockerfileSheet({
  closeSheet,
  ...props
}: DockerfileSheetProps) {
  const queryClient = useQueryClient();

  const form = useForm<DockerfileFormValues>({
    resolver: zodResolver(dockerfileFormSchema),
    defaultValues: {
      dockerfile:
        'FROM node:14\n\nWORKDIR /app\n\nCOPY package*.json ./\n\nRUN npm install\n\nCOPY . .\n\nEXPOSE 3000\n\nCMD ["npm", "start"]',
    },
  });

  const { mutate: submitDockerfileSheet, isPending } = useMutation({
    mutationFn: (values: DockerfileFormValues) =>
      apiUserCreateKaniko({
        description: values.description ? values.description : "",
        image: "",
        requirements: "",
        packages: "",
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ["imagelink", "list"],
      });
      closeSheet();
      toast.success(`镜像开始制作，请在下方列表中查看制作状态`);
    },
  });

  const onSubmit = (values: DockerfileFormValues) => {
    submitDockerfileSheet(values);
  };

  return (
    <Form {...form}>
      <SandwichSheet
        {...props}
        footer={
          <>
            <FormImportButton metadata={MetadataFormDockerfile} form={form} />
            <FormExportButton metadata={MetadataFormDockerfile} form={form} />
            <LoadableButton
              isLoading={isPending}
              isLoadingText="正在提交"
              type="submit"
              onClick={async () => {
                const isValid = await form.trigger();
                if (isValid) {
                  form.handleSubmit(onSubmit)();
                }
              }}
            >
              <PackagePlusIcon />
              开始制作
            </LoadableButton>
          </>
        }
      >
        <DockerfileSheetContent form={form} onSubmit={onSubmit} />
      </SandwichSheet>
    </Form>
  );
}

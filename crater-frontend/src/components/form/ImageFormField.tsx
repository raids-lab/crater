import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import FormLabelMust from "@/components/form/FormLabelMust";
import Combobox from "@/components/form/Combobox";
import ImageItem from "@/components/form/ImageItem";
import { FieldPath, FieldValues, UseFormReturn } from "react-hook-form";
import useImageQuery from "@/hooks/query/useImageQuery";
import { JobType } from "@/services/api/vcjob";

interface ImageFormFieldProps<T extends FieldValues> {
  form: UseFormReturn<T>;
  name: FieldPath<T>;
  jobType?: JobType;
  required?: boolean;
  label?: string;
  className?: string;
}

export function ImageFormField<T extends FieldValues>({
  form,
  name,
  jobType = JobType.Jupyter,
  required = true,
  label = "容器镜像",
  className,
}: ImageFormFieldProps<T>) {
  const { data: images } = useImageQuery(jobType);

  return (
    <FormField
      control={form.control}
      name={name}
      render={({ field }) => (
        <FormItem className={className}>
          <FormLabel>
            {label}
            {required && <FormLabelMust />}
          </FormLabel>
          <FormControl>
            <Combobox
              items={images ?? []}
              current={field.value}
              handleSelect={(value) => field.onChange(value)}
              renderLabel={(item) => <ImageItem item={item} />}
              formTitle="镜像"
            />
          </FormControl>
          <FormMessage />
        </FormItem>
      )}
    />
  );
}

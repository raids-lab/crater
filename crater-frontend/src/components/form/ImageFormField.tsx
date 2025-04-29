import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import FormLabelMust from "@/components/form/FormLabelMust";
import Combobox, { ComboboxItem } from "@/components/form/Combobox";
import ImageItem from "@/components/form/ImageItem";
import { FieldPath, FieldValues, UseFormReturn } from "react-hook-form";
import useImageQuery from "@/hooks/query/useImageQuery";
import { JobType } from "@/services/api/vcjob";
import { useCallback, useMemo, useState } from "react";
import { Badge } from "@/components/ui/badge";

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

export interface TagFilterProps {
  tags: string[];
  selectedTags: string[];
  onTagToggle: (tag: string) => void;
  className?: string;
  // label?: string;
}

export function TagFilter({
  tags,
  selectedTags,
  onTagToggle,
  className,
  // label = "按标签筛选",
}: TagFilterProps) {
  if (tags.length === 0) return null;

  return (
    <div className={`border-t border-b px-2 py-2 ${className}`}>
      {/* <div className="text-muted-foreground mb-1 text-xs">{label}</div> */}
      <div className="flex flex-wrap gap-1 pb-1">
        {tags.map((tag) => (
          <Badge
            key={tag}
            variant={selectedTags.includes(tag) ? "default" : "outline"}
            className="mb-1 cursor-pointer"
            onClick={(e) => {
              e.preventDefault();
              onTagToggle(tag);
            }}
          >
            {tag}
          </Badge>
        ))}
      </div>
    </div>
  );
}

// Hook for tag filtering logic
export function useTagFilter<T>(
  items: ComboboxItem<T>[],
  externalTags?: string[],
): {
  allTags: string[];
  selectedTags: string[];
  setSelectedTags: React.Dispatch<React.SetStateAction<string[]>>;
  toggleTag: (tag: string) => void;
  filterItemsByTags: (items: ComboboxItem<T>[]) => ComboboxItem<T>[];
} {
  const [selectedTags, setSelectedTags] = useState<string[]>([]);

  // Get all unique tags
  const allTags = useMemo(() => {
    if (externalTags && externalTags.length > 0) return externalTags;

    const tagSet = new Set<string>();
    items.forEach((item) => {
      if (item.tags && Array.isArray(item.tags)) {
        item.tags.forEach((tag) => tagSet.add(tag));
      }
    });
    return Array.from(tagSet);
  }, [items, externalTags]);

  // Toggle tag selection
  const toggleTag = useCallback((tag: string) => {
    setSelectedTags((prev) =>
      prev.includes(tag) ? prev.filter((t) => t !== tag) : [...prev, tag],
    );
  }, []);

  // Filter items based on selected tags
  const filterItemsByTags = useCallback(
    (itemsToFilter: ComboboxItem<T>[]) => {
      if (selectedTags.length === 0) return itemsToFilter;

      return itemsToFilter.filter((item) => {
        return (
          item.tags && selectedTags.every((tag) => item.tags?.includes(tag))
        );
      });
    },
    [selectedTags],
  );

  return {
    allTags,
    selectedTags,
    setSelectedTags,
    toggleTag,
    filterItemsByTags,
  };
}

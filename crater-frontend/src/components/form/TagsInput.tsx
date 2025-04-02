import * as React from "react";
import { ChevronsUpDown, XIcon } from "lucide-react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";

import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Input } from "@/components/ui/input";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { logger } from "@/utils/loglevel";

// Sample predefined tags
const predefinedTags = [
  { value: "react", label: "React" },
  { value: "nextjs", label: "Next.js" },
  { value: "tailwind", label: "Tailwind CSS" },
  { value: "ui", label: "UI" },
  { value: "design", label: "Design" },
  { value: "api", label: "API" },
  { value: "database", label: "Database" },
  { value: "javascript", label: "JavaScript" },
  { value: "typescript", label: "TypeScript" },
  { value: "node", label: "Node.js" },
  { value: "express", label: "Express" },
  { value: "graphql", label: "GraphQL" },
  { value: "redux", label: "Redux" },
  { value: "vue", label: "Vue.js" },
  { value: "angular", label: "Angular" },
  { value: "svelte", label: "Svelte" },
  { value: "docker", label: "Docker" },
  { value: "kubernetes", label: "Kubernetes" },
  { value: "aws", label: "AWS" },
  { value: "azure", label: "Azure" },
  { value: "gcp", label: "Google Cloud" },
  { value: "firebase", label: "Firebase" },
  { value: "mongodb", label: "MongoDB" },
  { value: "postgresql", label: "PostgreSQL" },
  { value: "mysql", label: "MySQL" },
  { value: "redis", label: "Redis" },
  { value: "testing", label: "Testing" },
  { value: "jest", label: "Jest" },
  { value: "cypress", label: "Cypress" },
  { value: "storybook", label: "Storybook" },
];

// Form schema
const formSchema = z.object({
  tags: z
    .array(
      z.object({
        value: z.string(),
        label: z.string(),
      }),
    )
    .min(1, "Please select at least one tag"),
});

type TagInputProps = {
  onSubmit?: (values: z.infer<typeof formSchema>) => void;
};

export default function TagInput({ onSubmit }: TagInputProps) {
  const [open, setOpen] = React.useState(false);
  const [inputValue, setInputValue] = React.useState("");

  // Initialize form
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      tags: [],
    },
  });

  const tags = form.watch("tags");

  // Handle form submission
  const handleSubmit = (values: z.infer<typeof formSchema>) => {
    logger.debug(values);
    if (onSubmit) {
      onSubmit(values);
    }
  };

  // Add a new custom tag
  const addCustomTag = () => {
    if (
      inputValue.trim() !== "" &&
      !tags.some((tag) => tag.value === inputValue.toLowerCase())
    ) {
      const newTag = {
        value: inputValue.toLowerCase(),
        label: inputValue,
      };
      form.setValue("tags", [...tags, newTag]);
      setInputValue("");
    }
  };

  // Remove a tag
  const removeTag = (tagToRemove: string) => {
    form.setValue(
      "tags",
      tags.filter((tag) => tag.value !== tagToRemove),
    );
  };

  // Handle key press in the input field
  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      e.preventDefault();
      if (inputValue.trim() !== "") {
        addCustomTag();
        // Remove this line to keep dropdown open
        // setOpen(false)
      }
    }
  };

  return (
    <div className="mx-auto w-full">
      <Form {...form}>
        <form onSubmit={form.handleSubmit(handleSubmit)} className="space-y-6">
          <FormField
            control={form.control}
            name="tags"
            render={() => (
              <FormItem>
                <FormLabel>标签</FormLabel>
                <FormControl className="flex flex-col gap-2">
                  <div className="flex gap-2">
                    {/* Dropdown for existing tags and adding new tags */}
                    <Popover modal open={open} onOpenChange={setOpen}>
                      <PopoverTrigger asChild>
                        <Button
                          variant={"outline"}
                          className={cn(
                            "flex h-full min-h-[36px] w-full cursor-pointer items-center justify-between px-3 py-1 font-normal whitespace-nowrap",
                            "data-[state=open]:border-ring data-[state=open]:ring-ring/50 data-[state=open]:ring-[3px]",
                          )}
                        >
                          <div
                            className={cn(
                              "items-center gap-1 overflow-hidden text-sm",
                              "flex grow flex-wrap",
                            )}
                          >
                            {tags.length > 0 ? (
                              <div className="flex flex-wrap gap-2">
                                {tags.map((tag) => (
                                  <span
                                    key={tag.value}
                                    className="bg-secondary text-secondary-foreground focus:ring-ring inline-flex items-center gap-1 rounded-md border py-0.5 pr-1 pl-2 text-xs font-medium transition-colors focus:ring-2 focus:ring-offset-2 focus:outline-hidden"
                                  >
                                    <span>{tag.label}</span>
                                    <span
                                      onClick={(e) => {
                                        e.preventDefault();
                                        removeTag(tag.value);
                                      }}
                                      className="text-muted-foreground/60 hover:bg-accent hover:text-muted-foreground flex cursor-pointer items-center rounded-sm px-[1px]"
                                    >
                                      <XIcon className="size-3" />
                                    </span>
                                  </span>
                                ))}
                              </div>
                            ) : (
                              <span className="text-muted-foreground mr-auto">
                                选择或添加标签
                              </span>
                            )}
                          </div>
                          <div className="text-muted-foreground hover:text-foreground flex items-center self-stretch pl-1 [&>div]:flex [&>div]:items-center [&>div]:self-stretch">
                            {tags.length > 0 ? (
                              <div
                                onClick={(e) => {
                                  e.preventDefault();
                                  form.setValue("tags", []);
                                }}
                              >
                                <XIcon className="size-4" />
                              </div>
                            ) : (
                              <div>
                                <ChevronsUpDown className="size-4 shrink-0 opacity-50" />
                              </div>
                            )}
                          </div>
                        </Button>
                      </PopoverTrigger>
                      <PopoverContent className="w-[var(--radix-popover-trigger-width)] p-0">
                        <Command>
                          <CommandInput placeholder="搜索标签..." />
                          <CommandList className="max-h-[200px] overflow-y-auto">
                            <CommandEmpty>未找到标签</CommandEmpty>
                            <CommandGroup heading="常用标签">
                              {predefinedTags
                                .filter(
                                  (tag) =>
                                    !tags.some((t) => t.value === tag.value),
                                )
                                .map((tag) => (
                                  <CommandItem
                                    key={tag.value}
                                    value={tag.value}
                                    onSelect={() => {
                                      form.setValue("tags", [...tags, tag]);
                                    }}
                                    className="cursor-pointer"
                                  >
                                    {tag.label}
                                  </CommandItem>
                                ))}
                            </CommandGroup>
                          </CommandList>
                          <div className="border-t p-2">
                            <div className="text-muted-foreground mb-2 ml-1 text-xs font-medium">
                              添加新标签
                            </div>
                            <div className="flex gap-2">
                              <Input
                                placeholder="输入新标签..."
                                value={inputValue}
                                onChange={(e) => setInputValue(e.target.value)}
                                onKeyDown={handleKeyDown}
                                className="flex-1"
                              />
                              <Button
                                type="button"
                                variant="secondary"
                                onClick={() => {
                                  addCustomTag();
                                  // Only close if explicitly requested by user
                                  // if (inputValue.trim() !== "") {
                                  //   setOpen(false)
                                  // }
                                }}
                                disabled={!inputValue.trim()}
                              >
                                添加
                              </Button>
                            </div>
                          </div>
                        </Command>
                      </PopoverContent>
                    </Popover>
                  </div>
                </FormControl>
                <FormDescription>选择已有标签或创建新标签</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </form>
      </Form>
    </div>
  );
}

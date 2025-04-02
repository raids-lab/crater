import { z } from "zod";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@/components/ui/button";
import { Form } from "@/components/ui/form";
import { TagsInput, Tag } from "@/components/form/TagsInput";
import { toast } from "sonner";
import { logger } from "@/utils/loglevel";

// 定义表单模式
const formSchema = z.object({
  name: z.string().min(1, "名称不能为空"),
  description: z.string().optional(),
  // 标签数组，至少需要一个标签
  tags: z
    .array(
      z.object({
        value: z.string(),
        label: z.string(),
      }),
    )
    .min(1, "请至少选择一个标签"),
});

// 表单数据类型
type FormValues = z.infer<typeof formSchema>;

// 自定义标签（可选）
const customTags: Tag[] = [
  { value: "python", label: "Python" },
  { value: "machine-learning", label: "机器学习" },
  { value: "deep-learning", label: "深度学习" },
  { value: "data-science", label: "数据科学" },
  { value: "nlp", label: "自然语言处理" },
];

export function CreateForm() {
  // 初始化表单
  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: "",
      description: "",
      tags: [],
    },
  });

  // 处理表单提交
  const onSubmit = (data: FormValues) => {
    toast.success("表单提交成功！");
    logger.debug(data);
    // 这里添加实际的提交逻辑
  };

  return (
    <div className="mx-auto w-full max-w-2xl p-4">
      <h1 className="mb-6 text-2xl font-bold">创建表单</h1>

      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
          {/* 其他表单字段 */}

          {/* 使用重构后的 TagsInput 组件 */}
          <TagsInput
            form={form}
            tagsPath="tags"
            label="项目标签"
            description="为项目添加标签以便分类和搜索"
            customTags={customTags}
            required={true}
          />

          <Button type="submit" className="w-full">
            提交
          </Button>
        </form>
      </Form>
    </div>
  );
}

import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Trash2, Edit, Plus, ExternalLink, CircleHelpIcon } from "lucide-react";
import { PodContainerDialogProps } from "@/components/codeblock/PodContainerDialog";
import { useNamespacedState } from "@/hooks/useNamespacedState";
import FormLabelMust from "@/components/custom/FormLabelMust";
import { toast } from "sonner";
import TooltipButton from "@/components/custom/TooltipButton";

const ingressFormSchema = z.object({
  id: z.string().optional(),
  // 只能包含小写字母，不超过20个字符。
  name: z
    .string()
    .min(1)
    .max(20)
    .regex(/^[a-z]+$/, {
      message: "只能包含小写字母",
    }),
  podPort: z.number().int().positive(),
});

type IngressRule = z.infer<typeof ingressFormSchema>;

type PodInfo = {
  ingress: IngressRule[];
};

export default function PodIngressDialog({
  namespacedName,
  setNamespacedName,
}: PodContainerDialogProps) {
  const [isOpen, setIsOpen] = useNamespacedState(
    namespacedName,
    setNamespacedName,
  );
  const [podInfo, setPodInfo] = useState<PodInfo>({
    ingress: [
      {
        id: "1",
        name: "notebook",
        podPort: 8888,
      },
    ],
  });

  const [isEditDialogOpen, setIsEditDialogOpen] = useState(false);

  const form = useForm<IngressRule>({
    resolver: zodResolver(ingressFormSchema),
    defaultValues: {
      name: "",
      podPort: 0,
    },
  });

  const handleEdit = (ingress: IngressRule) => {
    form.reset(ingress);
    setIsEditDialogOpen(true);
  };

  const handleDelete = (id: string) => {
    setPodInfo((prevState) => ({
      ...prevState,
      ingress: prevState.ingress.filter((ingress) => ingress.id !== id),
    }));
  };

  const handleAdd = () => {
    form.reset({
      id: "",
      name: "",
      podPort: 0,
    });
    setIsEditDialogOpen(true);
  };

  const onSubmit = (data: IngressRule) => {
    setPodInfo((prevState) => {
      const newIngress = prevState.ingress.filter((i) => i.id !== data.id);
      if (data.id) {
        newIngress.push(data);
      } else {
        newIngress.push({ ...data, id: Date.now().toString() });
      }
      return { ...prevState, ingress: newIngress };
    });
    setIsEditDialogOpen(false);
  };

  return (
    <Dialog open={isOpen} onOpenChange={setIsOpen}>
      <DialogContent className="sm:max-w-[625px]">
        <DialogHeader>
          <DialogTitle className="flex flex-row items-center justify-start">
            外部访问规则
            <CircleHelpIcon className="ml-1 h-4 w-4 text-muted-foreground" />
          </DialogTitle>
        </DialogHeader>
        <div className="grid gap-4 py-4">
          <div className="space-y-2">
            {podInfo.ingress.map((ingress) => (
              <div
                key={ingress.id}
                className="flex items-center space-x-2 rounded bg-secondary p-3"
              >
                <div className="ml-2 flex flex-grow flex-col items-start justify-start gap-0.5">
                  <p>{ingress.name}</p>
                  <div className="flex flex-row text-xs text-muted-foreground">
                    {ingress.podPort}
                    {" → crater.act.buaa.edu.cn/ingress/xxx"}
                  </div>
                </div>
                <TooltipButton
                  variant="ghost"
                  size="icon"
                  className="hover:text-primary"
                  onClick={() => toast.warning("功能开发中")}
                  tooltipContent="访问链接"
                >
                  <ExternalLink className="h-4 w-4" />
                </TooltipButton>
                <TooltipButton
                  variant="ghost"
                  size="icon"
                  className="hover:text-orange-500"
                  onClick={() => handleEdit(ingress)}
                  tooltipContent="编辑"
                >
                  <Edit className="h-4 w-4" />
                </TooltipButton>
                <TooltipButton
                  variant="ghost"
                  size="icon"
                  className="hover:text-destructive"
                  onClick={() => ingress.id && handleDelete(ingress.id)}
                  tooltipContent="删除"
                >
                  <Trash2 className="h-4 w-4" />
                </TooltipButton>
              </div>
            ))}
          </div>
        </div>
        <DialogFooter>
          {/* focus when open dialog */}
          <Button onClick={handleAdd} autoFocus>
            <Plus className="mr-2 h-4 w-4" />
            添加规则
          </Button>
        </DialogFooter>
      </DialogContent>

      <Dialog open={isEditDialogOpen} onOpenChange={setIsEditDialogOpen}>
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle>
              {form.getValues("id") ? "编辑" : "添加"}规则
            </DialogTitle>
          </DialogHeader>
          <Form {...form}>
            {/* eslint-disable-next-line @typescript-eslint/no-misused-promises */}
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
              <FormField
                control={form.control}
                name="name"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      规则名称
                      <FormLabelMust />
                    </FormLabel>
                    <FormControl>
                      <Input {...field} />
                    </FormControl>
                    <FormDescription>
                      只能包含小写字母，不超过20个字符。
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="podPort"
                render={() => (
                  <FormItem>
                    <FormLabel>
                      容器端口
                      <FormLabelMust />
                    </FormLabel>
                    <FormControl>
                      <Input
                        type="number"
                        {...form.register("podPort", {
                          valueAsNumber: true,
                        })}
                      />
                    </FormControl>
                    <FormDescription>
                      请输入容器内需要转发的端口号。
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <Button type="submit">保存</Button>
            </form>
          </Form>
        </DialogContent>
      </Dialog>
    </Dialog>
  );
}

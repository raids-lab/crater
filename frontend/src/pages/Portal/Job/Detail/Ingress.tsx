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
import { Trash2, Plus, ExternalLink, CircleHelpIcon } from "lucide-react";
import {
  NamespacedName,
  PodContainerDialogProps,
} from "@/components/codeblock/PodContainerDialog";
import { useNamespacedState } from "@/hooks/useNamespacedState";
import FormLabelMust from "@/components/custom/FormLabelMust";
import { toast } from "sonner";
import TooltipButton from "@/components/custom/TooltipButton";
import {
  apiGetPodIngresses,
  apiCreatePodIngress,
  apiDeletePodIngress,
} from "@/services/api/tool";
import {
  TooltipProvider,
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from "@/components/ui/tooltip";
import { useQuery } from "@tanstack/react-query";
import { PodIngress, PodIngressMgr } from "@/services/api/tool";

const ingressFormSchema = z.object({
  name: z
    .string()
    .min(1)
    .max(20)
    .regex(/^[a-z]+$/, {
      message: "只能包含小写字母",
    }),
  port: z.number().int().positive(),
});

type PodInfo = {
  ingress: PodIngress[];
};

export default function PodIngressDialog({
  namespacedName,
  setNamespacedName,
}: PodContainerDialogProps) {
  const [isOpen, setIsOpen] = useNamespacedState(
    namespacedName,
    setNamespacedName,
  );
  const [podInfo, setPodInfo] = useState<PodInfo>({ ingress: [] });
  const [isEditDialogOpen, setIsEditDialogOpen] = useState(false);

  const form = useForm<PodIngressMgr>({
    resolver: zodResolver(ingressFormSchema),
    defaultValues: {
      name: "",
      port: 0,
    },
  });

  const fetchIngresses = async (namespacedName: NamespacedName) => {
    if (!namespacedName) return [];
    const response = await apiGetPodIngresses(
      namespacedName.namespace,
      namespacedName.name,
    );
    return response.data.data.ingresses;
  };

  const { data: ingressData, refetch } = useQuery({
    queryKey: ["fetchIngresses", namespacedName],
    queryFn: () => fetchIngresses(namespacedName),
    select: (data) => data || [],
    enabled: !!namespacedName,
  });

  // 更新 podInfo
  if (ingressData && ingressData !== podInfo.ingress) {
    setPodInfo({ ingress: ingressData });
  }

  const handleAdd = () => {
    form.reset({ name: "", port: 0 });
    setIsEditDialogOpen(true);
  };

  const handleDelete = (data: PodIngressMgr) => {
    if (namespacedName) {
      apiDeletePodIngress(namespacedName.namespace, namespacedName.name, data)
        .then(() => {
          void refetch();
          toast.success("删除成功");
        })
        .catch(() => {});
    }
  };

  const onSubmit = (data: PodIngressMgr) => {
    if (namespacedName) {
      apiCreatePodIngress(namespacedName.namespace, namespacedName.name, data)
        .then(() => {
          void refetch();
          toast.success("添加成功");
          setIsEditDialogOpen(false);
        })
        .catch(() => {});
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={setIsOpen}>
      <DialogContent className="sm:max-w-[625px]">
        <DialogHeader>
          <DialogTitle className="flex flex-row items-center justify-start">
            外部访问规则
            <TooltipProvider delayDuration={10}>
              <Tooltip>
                <TooltipTrigger>
                  <CircleHelpIcon className="ml-1 h-4 w-4 text-muted-foreground" />
                </TooltipTrigger>
                <TooltipContent className="border bg-background text-foreground">
                  支持用户自定义 Ingress
                  <br />
                  将容器端口转发至外部访问
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          </DialogTitle>
        </DialogHeader>
        <div className="grid gap-4 py-4">
          <div className="space-y-2">
            {podInfo.ingress.map((ingress) => (
              <div
                key={ingress.name}
                className="flex items-center space-x-2 rounded bg-secondary p-3"
              >
                <div className="ml-2 flex flex-grow flex-col items-start justify-start gap-0.5">
                  <p>{ingress.name}</p>
                  <div className="flex flex-row text-xs text-muted-foreground">
                    {ingress.port}{" "}
                    {" → crater.act.buaa.edu.cn/ingress/" + ingress.prefix}
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
                  className="hover:text-destructive"
                  onClick={() => handleDelete(ingress)}
                  tooltipContent="删除"
                >
                  <Trash2 className="h-4 w-4" />
                </TooltipButton>
              </div>
            ))}
          </div>
        </div>
        <DialogFooter>
          <Button onClick={handleAdd} autoFocus>
            <Plus className="h-4 w-4" />
            添加规则
          </Button>
        </DialogFooter>
      </DialogContent>

      <Dialog open={isEditDialogOpen} onOpenChange={setIsEditDialogOpen}>
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle>添加规则</DialogTitle>
          </DialogHeader>
          <Form {...form}>
            <form
              onSubmit={(e) => {
                void form.handleSubmit(onSubmit)(e);
              }}
              className="space-y-4"
            >
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
                name="port"
                render={() => (
                  <FormItem>
                    <FormLabel>
                      容器端口
                      <FormLabelMust />
                    </FormLabel>
                    <FormControl>
                      <Input
                        type="number"
                        {...form.register("port", {
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

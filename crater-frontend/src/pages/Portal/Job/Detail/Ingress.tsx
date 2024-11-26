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
  apiGetPodNodeports,
  apiCreatePodNodeport,
  apiDeletePodNodeport,
} from "@/services/api/tool";
import {
  TooltipProvider,
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from "@/components/ui/tooltip";
import { useQuery } from "@tanstack/react-query";
import {
  PodIngress,
  PodIngressMgr,
  PodNodeport,
  PodNodeportMgr,
} from "@/services/api/tool";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui-custom/alert-dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

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

const nodeportFormSchema = z.object({
  name: z
    .string()
    .min(1)
    .max(20)
    .regex(/^[a-z]+$/, {
      message: "只能包含小写字母",
    }),
  containerPort: z.number().int().positive(),
});

type PodInfo = {
  ingress: PodIngress[];
  nodeport: PodNodeport[];
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
    ingress: [],
    nodeport: [],
  });
  const [isEditIngressDialogOpen, setIsEditIngressDialogOpen] = useState(false);
  const [isEditNodeportDialogOpen, setIsEditNodeportDialogOpen] =
    useState(false);
  const [activeTab, setActiveTab] = useState<"ingress" | "nodeport">("ingress"); // Tab状态

  const ingressForm = useForm<PodIngressMgr>({
    resolver: zodResolver(ingressFormSchema),
    defaultValues: {
      name: "",
      port: 0,
    },
  });

  const nodeportForm = useForm<PodNodeportMgr>({
    resolver: zodResolver(nodeportFormSchema),
    defaultValues: {
      name: "",
      containerPort: 0,
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

  const fetchNodeports = async (namespacedName: NamespacedName) => {
    if (!namespacedName) return [];
    const response = await apiGetPodNodeports(
      namespacedName.namespace,
      namespacedName.name,
    );
    return response.data.data.nodeports;
  };

  const { data: ingressData, refetch: refetchIngresses } = useQuery({
    queryKey: ["fetchIngresses", namespacedName],
    queryFn: () => fetchIngresses(namespacedName),
    select: (data) => data || [],
    enabled: !!namespacedName,
  });

  const { data: nodeportData, refetch: refetchNodeports } = useQuery({
    queryKey: ["fetchNodeports", namespacedName],
    queryFn: () => fetchNodeports(namespacedName),
    select: (data) => data || [],
    enabled: !!namespacedName,
  });

  // 更新 podInfo
  if (
    (ingressData && ingressData !== podInfo.ingress) ||
    (nodeportData && nodeportData !== podInfo.nodeport)
  ) {
    setPodInfo((prev) => ({
      ...prev,
      ingress: ingressData || [],
      nodeport: nodeportData || [],
    }));
  }

  const handleAddIngress = () => {
    ingressForm.reset({ name: "", port: 0 });
    setIsEditIngressDialogOpen(true);
  };

  const handleAddNodeport = () => {
    nodeportForm.reset({ name: "", containerPort: 0 });
    setIsEditNodeportDialogOpen(true);
  };

  const onSubmitIngress = (data: PodIngressMgr) => {
    if (namespacedName) {
      apiCreatePodIngress(
        namespacedName.namespace,
        namespacedName.name,
        data as PodIngressMgr,
      )
        .then(() => {
          void refetchIngresses(); // 刷新 Ingress 数据
          toast.success("添加成功");
          setIsEditIngressDialogOpen(false);
        })
        .catch(() => {
          toast.error("添加失败");
        });
    }
  };

  const onSubmitNodeport = (data: PodNodeportMgr) => {
    if (namespacedName) {
      apiCreatePodNodeport(
        namespacedName.namespace,
        namespacedName.name,
        data as PodNodeportMgr,
      )
        .then(() => {
          void refetchNodeports(); // 刷新 NodePort 数据
          toast.success("添加成功");
          setIsEditNodeportDialogOpen(false);
        })
        .catch(() => {
          toast.error("添加失败");
        });
    }
  };

  const handleDeleteIngress = (data: PodIngressMgr) => {
    if (namespacedName) {
      apiDeletePodIngress(namespacedName.namespace, namespacedName.name, data)
        .then(() => {
          void refetchIngresses();
          toast.success("删除成功");
        })
        .catch(() => {});
    }
  };

  const handleDeleteNodeport = (data: PodNodeportMgr) => {
    if (namespacedName) {
      apiDeletePodNodeport(namespacedName.namespace, namespacedName.name, data)
        .then(() => {
          void refetchNodeports(); // 刷新 NodePort 数据
          toast.success("删除成功");
        })
        .catch(() => {
          toast.error("删除失败");
        });
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

        {/* Tab内容 */}
        <Tabs
          defaultValue="ingress"
          value={activeTab}
          onValueChange={(value) =>
            setActiveTab(value as "ingress" | "nodeport")
          }
          className="w-full"
        >
          <TabsList className="flex border-b">
            <TabsTrigger value="ingress" className="flex-1 px-4 py-2">
              Ingress规则
            </TabsTrigger>
            <TabsTrigger value="nodeport" className="flex-1 px-4 py-2">
              NodePort规则
            </TabsTrigger>
          </TabsList>
          <TabsContent value="ingress" className="grid gap-4 py-4">
            <div className="space-y-2">
              {podInfo.ingress.map((ingress) => (
                <div
                  key={ingress.name}
                  className="flex items-center space-x-2 rounded bg-secondary p-3"
                >
                  <div className="ml-2 flex flex-grow flex-col items-start justify-start gap-0.5">
                    <p>{ingress.name}</p>
                    <div className="flex flex-row text-xs text-muted-foreground">
                      {ingress.port} → crater.act.buaa.edu.cn{ingress.prefix}
                    </div>
                  </div>
                  <TooltipButton
                    variant="ghost"
                    size="icon"
                    className="hover:text-primary"
                    onClick={() => {
                      const url =
                        "https://crater.act.buaa.edu.cn" + ingress.prefix;
                      window.open(url, "_blank");
                    }}
                    tooltipContent="访问链接"
                  >
                    <ExternalLink className="h-4 w-4" />
                  </TooltipButton>
                  <AlertDialog>
                    <AlertDialogTrigger asChild>
                      <TooltipButton
                        variant="ghost"
                        size="icon"
                        className="hover:text-destructive"
                        tooltipContent="删除"
                      >
                        <Trash2 className="h-4 w-4" />
                      </TooltipButton>
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>删除外部访问规则</AlertDialogTitle>
                        <AlertDialogDescription>
                          外部访问规则「{ingress.name}」<br />
                          {ingress.port} → crater.act.buaa.edu.cn
                          {ingress.prefix}
                          <br />
                          将被删除，请谨慎操作。
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel>取消</AlertDialogCancel>
                        <AlertDialogAction
                          variant="destructive"
                          onClick={() => handleDeleteIngress(ingress)}
                        >
                          删除
                        </AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                </div>
              ))}
            </div>
          </TabsContent>
          <TabsContent value="nodeport" className="grid gap-4 py-4">
            <div className="space-y-2">
              {podInfo.nodeport.map((nodeport) => (
                <div
                  key={nodeport.name}
                  className="flex items-center space-x-2 rounded bg-secondary p-3"
                >
                  <div className="ml-2 flex flex-grow flex-col items-start justify-start gap-0.5">
                    <p>{nodeport.name}</p>
                    <div className="flex flex-row text-xs text-muted-foreground">
                      {nodeport.containerPort} → {nodeport.address}:
                      {nodeport.nodePort}
                    </div>
                  </div>
                  <TooltipButton
                    variant="ghost"
                    size="icon"
                    className="hover:text-primary"
                    onClick={() => {
                      const url = `http://${nodeport.address}:${nodeport.nodePort}`;
                      window.open(url, "_blank");
                    }}
                    tooltipContent="访问链接"
                  >
                    <ExternalLink className="h-4 w-4" />
                  </TooltipButton>
                  <AlertDialog>
                    <AlertDialogTrigger asChild>
                      <TooltipButton
                        variant="ghost"
                        size="icon"
                        className="hover:text-destructive"
                        tooltipContent="删除"
                      >
                        <Trash2 className="h-4 w-4" />
                      </TooltipButton>
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>删除外部访问规则</AlertDialogTitle>
                        <AlertDialogDescription>
                          外部访问规则「{nodeport.name}」<br />
                          {nodeport.containerPort} → {nodeport.address}:
                          {nodeport.nodePort}
                          <br />
                          将被删除，请谨慎操作。
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel>取消</AlertDialogCancel>
                        <AlertDialogAction
                          variant="destructive"
                          onClick={() => handleDeleteNodeport(nodeport)}
                        >
                          删除
                        </AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                </div>
              ))}
            </div>
          </TabsContent>
        </Tabs>

        <DialogFooter>
          {activeTab === "ingress" && (
            <Button onClick={handleAddIngress} autoFocus>
              <Plus className="h-4 w-4" />
              添加 Ingress 规则
            </Button>
          )}
          {activeTab === "nodeport" && (
            <Button onClick={handleAddNodeport} autoFocus>
              <Plus className="h-4 w-4" />
              添加 NodePort 规则
            </Button>
          )}
        </DialogFooter>
      </DialogContent>

      <Dialog
        open={isEditIngressDialogOpen}
        onOpenChange={setIsEditIngressDialogOpen}
      >
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle>添加Ingress规则</DialogTitle>
          </DialogHeader>
          <Form {...ingressForm}>
            <form
              onSubmit={(e) => {
                void ingressForm.handleSubmit(onSubmitIngress)(e);
              }}
              className="space-y-4"
            >
              <FormField
                control={ingressForm.control}
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
                control={ingressForm.control}
                name="port"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      容器端口
                      <FormLabelMust />
                    </FormLabel>
                    <FormControl>
                      <Input
                        {...field}
                        type="number"
                        onChange={(e) => field.onChange(Number(e.target.value))} // 强制转换为数字
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
      <Dialog
        open={isEditNodeportDialogOpen}
        onOpenChange={setIsEditNodeportDialogOpen}
      >
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle>添加NodePort规则</DialogTitle>
          </DialogHeader>
          <Form {...nodeportForm}>
            <form
              onSubmit={async (e) => {
                void nodeportForm.handleSubmit(onSubmitNodeport)(e);
              }}
              className="space-y-4"
            >
              <FormField
                control={nodeportForm.control}
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
                      请为 NodePort 规则命名，不超过20个字符。
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={nodeportForm.control}
                name="containerPort"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      容器端口
                      <FormLabelMust />
                    </FormLabel>
                    <FormControl>
                      <Input
                        {...field}
                        type="number"
                        onChange={(e) => field.onChange(Number(e.target.value))} // 强制转换为数字
                      />
                    </FormControl>
                    <FormDescription>
                      请输入容器内需要使用的端口号。
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

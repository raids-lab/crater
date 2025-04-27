import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { useForm } from "react-hook-form";
import { useMutation, useQueryClient, useQuery } from "@tanstack/react-query";
import { toast } from "sonner";
import * as React from "react";
import { Button } from "@/components/ui/button";
import FormLabelMust from "@/components/form/FormLabelMust";
import {
  Resource,
  apiAdminResourceUpdate,
  apiResourceList,
  apiAdminResourceNetworkAdd,
  apiAdminResourceNetworksList,
  apiAdminResourceNetworkDelete,
} from "@/services/api/resource";
import { FC } from "react";
import { Badge } from "@/components/ui/badge";
import { PlusCircleIcon, XIcon } from "lucide-react";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

interface UpdateTaskFormProps extends React.HTMLAttributes<HTMLDivElement> {
  current: Resource;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const formSchema = z.object({
  id: z.number().int(),
  label: z.string().min(1, { message: "别名不能为空" }),
});

type FormSchema = z.infer<typeof formSchema>;

export function UpdateResourceForm({
  open,
  onOpenChange,
  current,
}: UpdateTaskFormProps) {
  const queryClient = useQueryClient();

  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      id: current.ID,
      label: current.label,
    },
  });

  const closeDialog = (event: { preventDefault: () => void }) => {
    event.preventDefault();
    onOpenChange(false);
  };

  const { mutate: updateLabelPack } = useMutation({
    mutationFn: (values: FormSchema) =>
      apiAdminResourceUpdate(values.id, values.label),
    onSuccess: async (_, { label }) => {
      await queryClient.invalidateQueries({
        queryKey: ["resource", "list"],
      });
      toast.success(`Label ${label} 更新成功`);
      onOpenChange(false);
    },
  });

  // 2. Define a submit handler.
  const onUpdateSubmit = (values: FormSchema) => {
    updateLabelPack(values);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>编辑资源</DialogTitle>
          <DialogDescription>{current.label} 的详细信息</DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form
            onSubmit={form.handleSubmit(onUpdateSubmit)}
            className="grid gap-4"
          >
            <FormField
              control={form.control}
              name="label"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    别名
                    <FormLabelMust />
                  </FormLabel>
                  <Input {...field} />
                  <FormDescription>
                    用于提交作业时，选择节点类型
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <DialogFooter className="grid grid-cols-2">
              <Button onClick={closeDialog} variant={"secondary"}>
                取消
              </Button>
              <Button type="submit">提交</Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}

interface UpdateResourceTypeFormProps {
  current: Resource;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const resourceTypeFormSchema = z.object({
  type: z.enum(["default", "gpu", "rdma"]),
});

export const UpdateResourceTypeForm: FC<UpdateResourceTypeFormProps> = ({
  current,
  open,
  onOpenChange,
}) => {
  const queryClient = useQueryClient();

  const form = useForm<z.infer<typeof resourceTypeFormSchema>>({
    resolver: zodResolver(resourceTypeFormSchema),
    defaultValues: {
      type: current.type || "default",
    },
  });

  const { mutate: updateType, isPending } = useMutation({
    mutationFn: (values: z.infer<typeof resourceTypeFormSchema>) => {
      return apiAdminResourceUpdate(current.ID, current.label, values.type);
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["resource", "list"] });
      await queryClient.invalidateQueries({
        queryKey: ["resource", "networks", current.ID],
      });
      toast.success("资源类型已更新");
      onOpenChange(false);
    },
  });

  function onSubmit(values: z.infer<typeof resourceTypeFormSchema>) {
    if (values.type === "default") {
      return;
    }
    updateType(values);
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>设置资源类型</DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="type"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>资源类型</FormLabel>
                  <Select
                    onValueChange={field.onChange}
                    defaultValue={field.value}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder="选择资源类型" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectItem value="default">无类型</SelectItem>
                      <SelectItem value="gpu">GPU</SelectItem>
                      <SelectItem value="rdma">RDMA</SelectItem>
                    </SelectContent>
                  </Select>
                  <FormDescription>
                    GPU 资源可以关联 RDMA 网络资源。
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <div className="flex justify-end">
              <Button type="submit" disabled={isPending}>
                保存
              </Button>
            </div>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
};

interface NetworkAssociationFormProps {
  gpuResource: Resource;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export const NetworkAssociationForm: FC<NetworkAssociationFormProps> = ({
  gpuResource,
  open,
  onOpenChange,
}) => {
  const queryClient = useQueryClient();

  // Fetch all RDMA resources
  const rdmaResourcesQuery = useQuery({
    queryKey: ["resource", "list", "rdma"],
    queryFn: () => apiResourceList(false),
    select: (res) => {
      return res.data.data.filter((r) => r.type === "rdma");
    },
  });

  // Fetch current network associations
  const currentNetworksQuery = useQuery({
    queryKey: ["resource", "networks", gpuResource.ID],
    queryFn: () => apiAdminResourceNetworksList(gpuResource.ID),
    select: (res) => res.data.data,
  });

  const { mutate: addNetworkAssociation, isPending } = useMutation({
    mutationFn: (rdmaId: number) => {
      return apiAdminResourceNetworkAdd(gpuResource.ID, rdmaId);
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ["resource", "networks", gpuResource.ID],
      });
      toast.success("网络关联已添加");
    },
  });

  const { mutate: removeNetworkAssociation } = useMutation({
    mutationFn: (rdmaId: number) =>
      apiAdminResourceNetworkDelete(gpuResource.ID, rdmaId),
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ["resource", "networks", gpuResource.ID],
      });
      toast.success("网络关联已删除");
    },
  });

  const currentNetworkIds = currentNetworksQuery.data?.map((n) => n.ID) || [];

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>管理网络关联</DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <div>
            <h3 className="mb-2 text-sm font-medium">GPU 资源</h3>
            <Badge className="font-mono" variant="secondary">
              {gpuResource.name}
            </Badge>
          </div>

          <div>
            <h3 className="mb-2 text-sm font-medium">当前关联的网络</h3>
            {currentNetworksQuery.isLoading ? (
              <div className="text-muted-foreground text-sm">加载中...</div>
            ) : currentNetworkIds.length === 0 ? (
              <div className="text-muted-foreground text-sm">暂无关联网络</div>
            ) : (
              <div className="flex flex-wrap gap-2">
                {currentNetworksQuery.data?.map((network) => (
                  <Badge
                    key={network.ID}
                    variant="outline"
                    className="font-mono"
                  >
                    {network.name}
                  </Badge>
                ))}
              </div>
            )}
          </div>

          <div>
            <h3 className="mb-2 text-sm font-medium">可用 RDMA 资源</h3>
            {rdmaResourcesQuery.isLoading ? (
              <div className="text-muted-foreground text-sm">加载中...</div>
            ) : rdmaResourcesQuery.data?.length === 0 ? (
              <div className="text-muted-foreground text-sm">
                暂无可用 RDMA 资源
              </div>
            ) : (
              <ScrollArea className="h-56 rounded-md border">
                <div className="space-y-2 p-4">
                  {rdmaResourcesQuery.data?.map((rdma) => {
                    const isAssociated = currentNetworkIds.includes(rdma.ID);

                    return (
                      <div
                        key={rdma.ID}
                        className="hover:bg-muted flex items-center justify-between rounded p-2"
                      >
                        <div className="flex items-center gap-2">
                          <Badge variant="outline" className="font-mono">
                            {rdma.name}
                          </Badge>
                          {rdma.label && (
                            <span className="text-muted-foreground text-xs">
                              {rdma.label}
                            </span>
                          )}
                        </div>

                        {isAssociated ? (
                          <Button
                            size="sm"
                            variant="destructive"
                            disabled={isPending}
                            onClick={() => removeNetworkAssociation(rdma.ID)}
                          >
                            <XIcon className="size-4" />
                            取消关联
                          </Button>
                        ) : (
                          <Button
                            size="sm"
                            variant="outline"
                            disabled={isPending}
                            onClick={() => addNetworkAssociation(rdma.ID)}
                          >
                            <PlusCircleIcon className="size-4" />
                            关联
                          </Button>
                        )}
                      </div>
                    );
                  })}
                </div>
              </ScrollArea>
            )}
          </div>

          <div className="flex justify-end pt-2">
            <Button variant="outline" onClick={() => onOpenChange(false)}>
              完成
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
};

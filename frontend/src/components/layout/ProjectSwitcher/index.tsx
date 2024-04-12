import {
  CaretSortIcon,
  CheckIcon,
  PlusCircledIcon,
} from "@radix-ui/react-icons";

import { cn } from "@/lib/utils";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from "@/components/ui/command";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { useEffect, useMemo, useState } from "react";
import { useRecoilState } from "recoil";
import { globalProject } from "@/utils/store";
import { ProjectBasic, ProjectStatus } from "@/services/api/project";
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
import { useForm } from "react-hook-form";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { logger } from "@/utils/loglevel";
import FormLabelMust from "@/components/custom/FormLabelMust";
import { Textarea } from "@/components/ui/textarea";
import { apiProjectCreate, apiProjectList } from "@/services/api/project";
import LoadableButton from "@/components/custom/LoadableButton";
import { apiProjectSwitch } from "@/services/api/auth";
import { useLocation } from "react-router-dom";

const formSchema = z.object({
  name: z
    .string()
    .min(1, {
      message: "项目名称不能为空",
    })
    .max(16, {
      message: "项目名称最多16个字符",
    }),
  description: z
    .string()
    .min(1, {
      message: "项目描述不能为空",
    })
    .max(170, {
      message: "项目描述最多170个字符",
    }),
  cpu: z.number().int().min(0, {
    message: "如需取消上限，请联系管理员",
  }),
  gpu: z.number().int().min(0, {
    message: "如需取消上限，请联系管理员",
  }),
  memory: z.number().int().min(0, {
    message: "如需取消上限，请联系管理员",
  }),
  storage: z.number().int().min(0, {
    message: "如需取消上限，请联系管理员",
  }),
});

interface ProjectGroup {
  personal: ProjectBasic[];
  team: ProjectBasic[];
}

type PopoverTriggerProps = React.ComponentPropsWithoutRef<
  typeof PopoverTrigger
>;

interface ProjectSwitcherProps extends PopoverTriggerProps {
  className?: string;
}

export default function ProjectSwitcher({ className }: ProjectSwitcherProps) {
  const [open, setOpen] = useState(false);
  const [showNewTeamDialog, setShowNewTeamDialog] = useState(false);
  const [selectedProject, setSelectedProject] = useRecoilState(globalProject);
  const queryClient = useQueryClient();
  const location = useLocation();

  const isAdminView = useMemo(() => {
    const pathParts = location.pathname.split("/").filter(Boolean);
    return pathParts[0] === "admin";
  }, [location]);

  const { data: projects } = useQuery({
    queryKey: ["user", "projects"],
    queryFn: apiProjectList,
    select: (res) => res.data.data,
  });

  const { mutate: createNewProject, isPending } = useMutation({
    mutationFn: (values: z.infer<typeof formSchema>) =>
      apiProjectCreate({
        name: values.name,
        description: values.description,
        quota: {
          cpu: values.cpu,
          gpu: values.gpu,
          memory: values.memory,
          storage: values.storage,
        },
      }),
    onSuccess: (_, { name }) => {
      void queryClient.invalidateQueries({ queryKey: ["user", "projects"] });
      toast.success(`已提交项目 ${name} 申请，请等待管理员审核`);
      setShowNewTeamDialog(false);
      form.reset();
    },
  });

  const { mutate: switchProject } = useMutation({
    mutationFn: (project: ProjectBasic) => apiProjectSwitch(project.id),
    onSuccess: (_, project) => {
      setSelectedProject(project);
      setOpen(false);
      toast.success(`已切换至项目 ${project.name}`);
      void queryClient.invalidateQueries();
    },
  });

  // 1. Define your form.
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: "",
      description: "",
      cpu: 0,
      gpu: 0,
      memory: 0,
      storage: 0,
    },
  });

  // 2. Define a submit handler.
  const onSubmit = (values: z.infer<typeof formSchema>) => {
    // Do something with the form values.
    // ✅ This will be type-safe and validated.
    !isPending && createNewProject(values);
  };

  const projectGroup: ProjectGroup = useMemo(() => {
    return {
      personal:
        projects?.filter((project) => project.isPersonal === true) ?? [],
      team:
        projects
          ?.filter((project) => project.isPersonal === false)
          .sort((a, b) => b.status - a.status) ?? [],
    };
  }, [projects]);

  useEffect(() => {
    if (!selectedProject.id) {
      setSelectedProject(projectGroup.personal[0]);
    }
  }, [selectedProject, setSelectedProject, projectGroup.personal]);

  return (
    <div className={className}>
      <Dialog open={showNewTeamDialog} onOpenChange={setShowNewTeamDialog}>
        <Popover open={open} onOpenChange={setOpen}>
          <PopoverTrigger asChild>
            <Button
              variant="outline"
              role="combobox"
              aria-expanded={open}
              aria-label="Select a project"
              className="w-full justify-between rounded-lg bg-background md:w-[240px]"
              disabled={isAdminView}
            >
              <Avatar className="mr-2 h-5 w-5">
                <AvatarFallback className="bg-primary font-normal text-primary-foreground">
                  {selectedProject.name.slice(0, 1).toUpperCase()}
                </AvatarFallback>
              </Avatar>
              <p className="capitalize">{selectedProject.name}</p>
              <CaretSortIcon className="ml-auto h-4 w-4 shrink-0 opacity-50" />
            </Button>
          </PopoverTrigger>
          <PopoverContent
            className="p-0"
            style={{
              width: "var(--radix-popover-trigger-width)",
              maxHeight: "var(--radix-popover-content-available-height)",
            }}
          >
            <Command>
              <CommandList>
                <CommandInput placeholder="查找项目" />
                <CommandEmpty>暂无数据</CommandEmpty>
                {projectGroup.personal.length > 0 && (
                  <CommandGroup heading={"个人项目"}>
                    {projectGroup.personal.map((team) => (
                      <CommandItem
                        key={`team-${team.id}`}
                        onSelect={() => {
                          switchProject(team);
                        }}
                        className="text-sm"
                      >
                        <Avatar className="mr-2 h-5 w-5">
                          <AvatarFallback className="bg-primary/15">
                            {team.name.slice(0, 1).toUpperCase()}
                          </AvatarFallback>
                        </Avatar>
                        <p className="capitalize">{team.name}</p>
                        <CheckIcon
                          className={cn(
                            "ml-auto h-4 w-4",
                            selectedProject.id === team.id
                              ? "opacity-100"
                              : "opacity-0",
                          )}
                        />
                      </CommandItem>
                    ))}
                  </CommandGroup>
                )}
                {projectGroup.team.length > 0 && (
                  <CommandGroup heading={"团队项目"}>
                    {projectGroup.team.map((team) => (
                      <CommandItem
                        key={`team-${team.id}`}
                        onSelect={() => {
                          switchProject(team);
                        }}
                        className="text-sm"
                        disabled={team.status !== ProjectStatus.Active}
                      >
                        <Avatar className="mr-2 h-5 w-5">
                          <AvatarFallback className="bg-primary/15">
                            {team.name.slice(0, 1).toUpperCase()}
                          </AvatarFallback>
                        </Avatar>
                        <p className="capitalize">{team.name}</p>
                        <CheckIcon
                          className={cn(
                            "ml-auto h-4 w-4",
                            selectedProject.id === team.id
                              ? "opacity-100"
                              : "opacity-0",
                          )}
                        />
                      </CommandItem>
                    ))}
                  </CommandGroup>
                )}
              </CommandList>
              <CommandSeparator />
              <CommandList>
                <CommandGroup>
                  <DialogTrigger asChild>
                    <CommandItem
                      onSelect={() => {
                        setOpen(false);
                        setShowNewTeamDialog(true);
                      }}
                    >
                      <PlusCircledIcon className="mr-2" />
                      新建团队项目
                    </CommandItem>
                  </DialogTrigger>
                </CommandGroup>
              </CommandList>
            </Command>
          </PopoverContent>
        </Popover>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>新建团队项目</DialogTitle>
            <DialogDescription>
              团队项目可包含多名成员，成员间共享配额、镜像、数据等资源。
            </DialogDescription>
          </DialogHeader>
          <Form {...form}>
            <form
              // eslint-disable-next-line @typescript-eslint/no-misused-promises
              onSubmit={form.handleSubmit(onSubmit)}
              className="grid gap-4 md:grid-cols-2"
            >
              <FormField
                control={form.control}
                name="name"
                render={({ field }) => (
                  <FormItem className="col-span-2">
                    <FormLabel>
                      项目名称
                      <FormLabelMust />
                    </FormLabel>
                    <FormControl>
                      <Input autoComplete="off" {...field} />
                    </FormControl>
                    <FormDescription>
                      名称是团队项目的唯一标识，最多16个字符
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="description"
                render={({ field }) => (
                  <FormItem className="col-span-2">
                    <FormLabel>
                      项目描述
                      <FormLabelMust />
                    </FormLabel>
                    <FormControl>
                      <Textarea {...field} />
                    </FormControl>
                    <FormDescription>
                      项目描述将是管理员审批的重要依据，最多170个字符
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="cpu"
                render={() => (
                  <FormItem>
                    <FormLabel>
                      申请 CPU 核心数
                      <FormLabelMust />
                    </FormLabel>
                    <FormControl>
                      <Input
                        type="number"
                        {...form.register("cpu", { valueAsNumber: true })}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="gpu"
                render={() => (
                  <FormItem>
                    <FormLabel>
                      申请 GPU 卡数
                      <FormLabelMust />
                    </FormLabel>
                    <FormControl>
                      <Input
                        type="number"
                        {...form.register("gpu", { valueAsNumber: true })}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="memory"
                render={() => (
                  <FormItem>
                    <FormLabel>
                      内存上限 (GB)
                      <FormLabelMust />
                    </FormLabel>
                    <FormControl>
                      <Input
                        type="number"
                        {...form.register("memory", { valueAsNumber: true })}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="storage"
                render={() => (
                  <FormItem>
                    <FormLabel>
                      项目存储空间 (GB)
                      <FormLabelMust />
                    </FormLabel>
                    <FormControl>
                      <Input
                        type="number"
                        {...form.register("storage", { valueAsNumber: true })}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </form>
          </Form>
          <DialogFooter>
            <LoadableButton
              isLoading={isPending}
              type="submit"
              onClick={() => {
                form
                  .trigger()
                  .then(() => {
                    if (form.formState.isValid) {
                      onSubmit(form.getValues());
                    }
                  })
                  .catch((e) => logger.debug(e));
              }}
            >
              提交申请
            </LoadableButton>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

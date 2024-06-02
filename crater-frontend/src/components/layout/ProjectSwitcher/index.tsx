import { CaretSortIcon, CheckIcon } from "@radix-ui/react-icons";

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
} from "@/components/ui/command";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { useMemo, useState } from "react";
import { useRecoilState } from "recoil";
import { globalAccount } from "@/utils/store";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { apiQueueSwitch } from "@/services/api/auth";
import { useLocation } from "react-router-dom";
import { QueueBasic, apiQueueList } from "@/services/api/queue";

type PopoverTriggerProps = React.ComponentPropsWithoutRef<
  typeof PopoverTrigger
>;

interface ProjectSwitcherProps extends PopoverTriggerProps {
  className?: string;
}

export default function ProjectSwitcher({ className }: ProjectSwitcherProps) {
  const [open, setOpen] = useState(false);
  const [account, setAccount] = useRecoilState(globalAccount);
  const queryClient = useQueryClient();
  const location = useLocation();

  const isAdminView = useMemo(() => {
    const pathParts = location.pathname.split("/").filter(Boolean);
    return pathParts[0] === "admin";
  }, [location]);

  const { data: projects } = useQuery({
    queryKey: ["queues"],
    queryFn: apiQueueList,
    select: (res) => res.data.data,
  });

  const currentQueue = useMemo(() => {
    return projects?.find((p) => p.id === account.queue);
  }, [projects, account.queue]);

  const { mutate: switchQueue } = useMutation({
    mutationFn: (project: QueueBasic) => apiQueueSwitch(project.id),
    onSuccess: ({ context }, { name }) => {
      setAccount(context);
      setOpen(false);
      toast.success(`已切换至账户 ${name}`);
      void queryClient.invalidateQueries();
    },
  });

  return (
    <div className={className}>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            variant="outline"
            role="combobox"
            aria-expanded={open}
            aria-label="Select a project"
            className={cn(
              "w-full justify-between bg-background px-2 md:w-[240px]",
              {
                hidden: isAdminView,
              },
            )}
          >
            {currentQueue && currentQueue.name !== "" ? (
              <>
                <Avatar className="mr-2 h-5 w-5">
                  <AvatarFallback className="bg-primary/15 font-normal text-primary">
                    {currentQueue.name.slice(0, 1).toUpperCase()}
                  </AvatarFallback>
                </Avatar>
                <p className="capitalize">{currentQueue.name}</p>
              </>
            ) : (
              <>
                <Avatar className="mr-2 h-5 w-5">
                  <AvatarFallback className="bg-primary/15 font-normal text-primary" />
                </Avatar>
                <p className="text-muted-foreground">未选择账户</p>
              </>
            )}
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
              <CommandInput placeholder="查找账户" />
              <CommandEmpty>暂无数据</CommandEmpty>
              {!!projects && projects.length > 0 && (
                <CommandGroup>
                  {projects.map((team) => (
                    <CommandItem
                      key={`team-${team.id}`}
                      onSelect={() => {
                        if (currentQueue?.id !== team.id) {
                          switchQueue(team);
                        } else {
                          setOpen(false);
                        }
                      }}
                      className="text-sm"
                    >
                      <Avatar className="mr-2 h-5 w-5">
                        <AvatarFallback className="bg-primary font-normal text-primary-foreground">
                          {team.name.slice(0, 1).toUpperCase()}
                        </AvatarFallback>
                      </Avatar>
                      <p className="capitalize">{team.name}</p>
                      <CheckIcon
                        className={cn(
                          "ml-auto h-4 w-4",
                          currentQueue?.id === team.id
                            ? "opacity-100"
                            : "opacity-0",
                        )}
                      />
                    </CommandItem>
                  ))}
                </CommandGroup>
              )}
            </CommandList>
          </Command>
        </PopoverContent>
      </Popover>
    </div>
  );
}

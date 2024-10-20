import { ChevronsUpDown } from "lucide-react";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from "@/components/ui/sidebar";
import { useMemo } from "react";
import { globalAccount } from "@/utils/store";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { apiQueueSwitch } from "@/services/api/auth";
import { useLocation } from "react-router-dom";
import { QueueBasic, apiQueueList } from "@/services/api/queue";
import { useAtom } from "jotai";
import { showErrorToast } from "@/utils/toast";
import { Avatar } from "@radix-ui/react-avatar";
import Identicon from "@polkadot/react-identicon";
import { stringToSS58 } from "@/utils/ss58";

export function TeamSwitcher() {
  const { isMobile } = useSidebar();

  const [account, setAccount] = useAtom(globalAccount);
  const queryClient = useQueryClient();
  const location = useLocation();

  const isAdminView = useMemo(() => {
    const pathParts = location.pathname.split("/").filter(Boolean);
    return pathParts[0] === "admin";
  }, [location]);

  const { data: queues } = useQuery({
    queryKey: ["queues"],
    queryFn: apiQueueList,
    select: (res) => res.data.data,
  });

  const currentQueue = useMemo(() => {
    return queues?.find((p) => p.id === account.queue);
  }, [queues, account]);

  const { mutate: switchQueue } = useMutation({
    mutationFn: (project: QueueBasic) => apiQueueSwitch(project.id),
    onSuccess: ({ context }, { name }) => {
      setAccount(context);
      toast.success(`已切换至账户 ${name}`);
      void queryClient.invalidateQueries();
    },
    onError: (error) => {
      showErrorToast(error);
    },
  });

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger disabled={isAdminView} asChild>
            <SidebarMenuButton
              size="lg"
              className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
            >
              <Avatar className="h-8 w-8 rounded-full">
                {currentQueue?.id && (
                  <Identicon
                    value={stringToSS58(currentQueue?.id)}
                    size={32}
                    // 'beachball' | 'empty' | 'ethereum' | 'jdenticon' | 'polkadot' | 'substrate'
                    theme="substrate"
                    className="!cursor-pointer"
                  />
                )}
              </Avatar>
              <div className="grid flex-1 text-left text-sm leading-tight">
                <span className="truncate font-semibold">
                  {currentQueue?.name}
                </span>
                <span className="truncate text-xs">{currentQueue?.id}</span>
              </div>
              <ChevronsUpDown className="ml-auto" />
            </SidebarMenuButton>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            className="w-[--radix-dropdown-menu-trigger-width] min-w-44 rounded-lg"
            align="start"
            side={isMobile ? "bottom" : "right"}
            sideOffset={4}
          >
            <DropdownMenuLabel className="text-xs text-muted-foreground">
              资源组
            </DropdownMenuLabel>
            {queues?.map((queue) => (
              <DropdownMenuItem
                key={queue.name}
                onClick={() => {
                  if (currentQueue?.id !== queue.id) {
                    switchQueue(queue);
                  }
                }}
                className="gap-2 p-2"
              >
                <div className="size-6 overflow-hidden">
                  <Identicon
                    value={stringToSS58(queue.id)}
                    size={24}
                    theme="substrate"
                    className="!cursor-pointer"
                  />
                </div>
                <span className="truncate">{queue.name}</span>
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  );
}

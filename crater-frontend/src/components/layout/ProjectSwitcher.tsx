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
import { Label } from "@/components/ui/label";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useEffect, useMemo, useState } from "react";
import { useRecoilState, useRecoilValue } from "recoil";
import { globalCurrentAccount, globalUserInfo } from "@/utils/store";
import { ProjectStatus } from "@/services/api/auth";

interface Project {
  label: string;
  id: number;
  enabled: boolean;
}

interface ProjectGroup {
  personal: Project[];
  team: Project[];
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
  const [selectedTeam, setSelectedTeam] = useRecoilState(globalCurrentAccount);
  const { projects } = useRecoilValue(globalUserInfo);

  const projectGroup: ProjectGroup = useMemo(() => {
    const personal = projects.find((project) => project.isPersonal === true);
    return {
      personal: [
        {
          label: personal?.name ?? "Personal",
          id: personal?.id ?? 0,
          enabled: true,
        },
      ],
      team: [
        {
          id: 4,
          name: "Team 4",
          role: 1,
          isPersonal: false,
          status: ProjectStatus.Pending,
        },
        {
          id: 2,
          name: "Team 1",
          role: 1,
          isPersonal: false,
          status: ProjectStatus.Active,
        },
        {
          id: 3,
          name: "Team 2",
          role: 1,
          isPersonal: false,
          status: ProjectStatus.Pending,
        },
        ...projects,
      ]
        .filter((project) => project.isPersonal === false)
        .sort((a, b) => b.status - a.status)
        .map((project) => ({
          label: project.name,
          id: project.id,
          enabled: project.status === ProjectStatus.Active,
        })),
    };
  }, [projects]);

  useEffect(() => {
    if (!selectedTeam.id) {
      setSelectedTeam(projectGroup.personal[0]);
    }
  }, [selectedTeam, setSelectedTeam, projectGroup.personal]);

  return (
    <div className={className}>
      <Dialog open={showNewTeamDialog} onOpenChange={setShowNewTeamDialog}>
        <Popover open={open} onOpenChange={setOpen}>
          <PopoverTrigger asChild>
            <Button
              variant="outline"
              role="combobox"
              aria-expanded={open}
              aria-label="Select a team"
              className={cn(
                "w-full rounded-lg bg-background md:w-[240px]",
                "justify-between",
              )}
            >
              <Avatar className="mr-2 h-5 w-5">
                <AvatarFallback className="bg-primary font-normal text-primary-foreground">
                  {selectedTeam.label.slice(0, 1).toUpperCase()}
                </AvatarFallback>
              </Avatar>
              <p className="capitalize">{selectedTeam.label}</p>
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
                          setSelectedTeam(team);
                          setOpen(false);
                        }}
                        className="text-sm"
                      >
                        <Avatar className="mr-2 h-5 w-5">
                          <AvatarFallback className="bg-primary/15">
                            {team.label.slice(0, 1).toUpperCase()}
                          </AvatarFallback>
                        </Avatar>
                        <p className="capitalize">{team.label}</p>
                        <CheckIcon
                          className={cn(
                            "ml-auto h-4 w-4",
                            selectedTeam.id === team.id
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
                          setSelectedTeam(team);
                          setOpen(false);
                        }}
                        className="text-sm"
                        disabled={!team.enabled}
                      >
                        <Avatar className="mr-2 h-5 w-5">
                          <AvatarFallback className="bg-primary/15">
                            {team.label.slice(0, 1).toUpperCase()}
                          </AvatarFallback>
                        </Avatar>
                        <p className="capitalize">{team.label}</p>
                        <CheckIcon
                          className={cn(
                            "ml-auto h-4 w-4",
                            selectedTeam.id === team.id
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
              创建一个新的团队项目，邀请成员一起协作。
            </DialogDescription>
          </DialogHeader>
          <div>
            <div className="space-y-4 py-2 pb-4">
              <div className="space-y-2">
                <Label htmlFor="name">Team name</Label>
                <Input id="name" placeholder="Acme Inc." />
              </div>
              <div className="space-y-2">
                <Label htmlFor="plan">Subscription plan</Label>
                <Select>
                  <SelectTrigger>
                    <SelectValue placeholder="Select a plan" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="free">
                      <span className="font-medium">Free</span> -{" "}
                      <span className="text-muted-foreground">
                        Trial for two weeks
                      </span>
                    </SelectItem>
                    <SelectItem value="pro">
                      <span className="font-medium">Pro</span> -{" "}
                      <span className="text-muted-foreground">
                        $9/month per user
                      </span>
                    </SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowNewTeamDialog(false)}
            >
              Cancel
            </Button>
            <Button type="submit">Continue</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

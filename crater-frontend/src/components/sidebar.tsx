import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";

export const playlists = ["训练任务列表", "新建训练任务", "任务监控"];

type Playlist = (typeof playlists)[number];

interface SidebarProps extends React.HTMLAttributes<HTMLDivElement> {
  playlists: Playlist[];
}

export function Sidebar({ className, playlists }: SidebarProps) {
  return (
    <ScrollArea className={cn("h-screen", className)}>
      <div className="py-4">
        <Accordion
          type="multiple"
          // collapsible
          className="px-2 py-2"
          defaultValue={["item-2"]}
        >
          {[
            "集群资源查看",
            "训练任务管理",
            "资源配额",
            "Jupyter 管理",
            "Scroll Test",
          ].map((name, i) => (
            <AccordionItem value={`item-${i + 1}`} key={`sidebar-item-${i}`}>
              <AccordionTrigger>
                <div
                  key={`$button-${i}`}
                  className="flex flex-row items-center justify-start"
                >
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    className="mr-2 h-4 w-4"
                  >
                    <rect width="7" height="7" x="3" y="3" rx="1" />
                    <rect width="7" height="7" x="14" y="3" rx="1" />
                    <rect width="7" height="7" x="14" y="14" rx="1" />
                    <rect width="7" height="7" x="3" y="14" rx="1" />
                  </svg>
                  {name}
                </div>
              </AccordionTrigger>
              <AccordionContent>
                {playlists?.map((playlist, ii) => (
                  <Button
                    key={`${name}-${ii}`}
                    variant="ghost"
                    className="w-full justify-start pl-10"
                  >
                    {playlist}
                  </Button>
                ))}
              </AccordionContent>
            </AccordionItem>
          ))}
        </Accordion>
      </div>
    </ScrollArea>
  );
}

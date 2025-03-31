import TipBadge from "@/components/badge/TipBadge";
import { TimeDistance } from "../TimeDistance";
import {
  Timeline,
  TimelineItem,
  TimelineConnector,
  TimelineHeader,
  TimelineTitle,
  TimelineIcon,
  TimelineDescription,
  TimelineContent,
  TimelineTime,
} from "./timeline";
import { Event as KubernetesEvent } from "kubernetes-types/core/v1";
import { cn } from "@/lib/utils";

interface TimelineLayoutProps {
  items: KubernetesEvent[]; // Replace any[] with the actual type of items.
}

export const EventTimeline = ({ items }: TimelineLayoutProps) => {
  return (
    <Timeline className="pt-4">
      {items
        .sort((a, b) => {
          if (!a.lastTimestamp) {
            return -1;
          }
          if (!b.lastTimestamp) {
            return 1;
          }
          return (
            new Date(b.lastTimestamp).getTime() -
            new Date(a.lastTimestamp).getTime()
          );
        })
        .map((item, index) => (
          <TimelineItem key={index} className="ml-24">
            {index < items.length - 1 && <TimelineConnector />}
            <TimelineHeader>
              <TimelineTime>
                <TimeDistance
                  date={item.lastTimestamp}
                  className="text-muted-foreground"
                />
              </TimelineTime>
              <TimelineIcon />
              <TimelineTitle className="flex flex-row items-center gap-1.5">
                <TipBadge
                  title={item.type}
                  className={cn({
                    "bg-secondary text-secondary-foreground hover:bg-accent capitalize":
                      item.type === "Normal",
                  })}
                />
                {item.involvedObject.kind} / {item.reason}
              </TimelineTitle>
            </TimelineHeader>
            <TimelineContent>
              <TimelineDescription>{item.message}</TimelineDescription>
            </TimelineContent>
          </TimelineItem>
        ))}
    </Timeline>
  );
};

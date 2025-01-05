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

interface TimelineLayoutProps {
  items: KubernetesEvent[]; // Replace any[] with the actual type of items.
}

export const EventTimeline = ({ items }: TimelineLayoutProps) => {
  return (
    <Timeline>
      {items
        .sort((a, b) => {
          if (!a.lastTimestamp) {
            return 1;
          }
          if (!b.lastTimestamp) {
            return -1;
          }
          return (
            new Date(a.lastTimestamp).getTime() -
            new Date(b.lastTimestamp).getTime()
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
                  className="bg-secondary capitalize text-secondary-foreground hover:bg-accent"
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

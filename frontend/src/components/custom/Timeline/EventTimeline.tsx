import TipBadge from "@/components/badge/TipBadge";
import { TimeDistance } from "../TimeDistance";
import {
  Timeline,
  TimelineItem,
  TimelineConnector,
  TimelineHeader,
  TimelineIcon,
  TimelineTitle,
  TimelineBody,
} from "@/components/custom/Timeline/Timeline";
import { Event as KubernetesEvent } from "kubernetes-types/core/v1";
import { cn } from "@/lib/utils";

interface TimelineLayoutProps {
  items: KubernetesEvent[]; // Replace any[] with the actual type of items.
}

export const EventTimeline = ({ items }: TimelineLayoutProps) => {
  return (
    <Timeline>
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
          <TimelineItem key={index}>
            {index < items.length - 1 && <TimelineConnector />}
            <TimelineHeader>
              <TimelineIcon>
                <div className="bg-primary h-2 w-2 rounded-full" />
              </TimelineIcon>
              <TimelineTitle>
                {item.reason && <>{item.reason}</>}
                {item.lastTimestamp && (
                  <span className="">
                    <span className="text-primary mx-1">-</span>
                    <TimeDistance date={item.lastTimestamp} />
                  </span>
                )}
              </TimelineTitle>
            </TimelineHeader>
            <TimelineBody>
              <div className="mb-2 flex flex-row items-center gap-1.5">
                <TipBadge
                  title={item.type}
                  className={cn({
                    "bg-secondary text-secondary-foreground hover:bg-accent capitalize":
                      item.type === "Normal",
                  })}
                />
                {item.involvedObject.kind && (
                  <TipBadge
                    title={item.involvedObject.kind}
                    className={cn({
                      "bg-secondary text-secondary-foreground hover:bg-accent capitalize":
                        item.type === "Normal",
                    })}
                  />
                )}
                {item.involvedObject.name && (
                  <TipBadge
                    title={item.involvedObject.name}
                    className={cn({
                      "bg-secondary text-secondary-foreground hover:bg-accent":
                        item.type === "Normal",
                    })}
                  />
                )}
              </div>
              <p className="text-muted-foreground font-mono text-sm">
                {item.message}
              </p>
            </TimelineBody>
          </TimelineItem>
        ))}
    </Timeline>
  );
};

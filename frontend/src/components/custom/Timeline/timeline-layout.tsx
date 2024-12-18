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

export interface TimelineElement {
  id: number;
  title: string;
  date: string;
  description: string;
}

interface TimelineLayoutProps {
  items: TimelineElement[]; // Replace any[] with the actual type of items.
}

export const TimelineLayout = ({ items }: TimelineLayoutProps) => {
  return (
    <Timeline>
      {items.map((item, index) => (
        <TimelineItem key={item.id} className="md:ml-16">
          {index < items.length - 1 && <TimelineConnector />}
          <TimelineHeader>
            <TimelineTime>
              <TimeDistance date={item.date} />
            </TimelineTime>
            <TimelineIcon />
            <TimelineTitle>{item.title}</TimelineTitle>
          </TimelineHeader>
          <TimelineContent>
            <TimelineDescription>{item.description}</TimelineDescription>
          </TimelineContent>
        </TimelineItem>
      ))}
    </Timeline>
  );
};

import * as React from "react";
import { cn } from "@/lib/utils";

const Timeline = ({
  ref,
  className,
  ...props
}: React.HTMLAttributes<HTMLOListElement> & {
  ref?: React.RefObject<HTMLOListElement>;
}) => <ol ref={ref} className={cn("flex flex-col", className)} {...props} />;
Timeline.displayName = "Timeline";

const TimelineItem = ({
  ref,
  className,
  ...props
}: React.LiHTMLAttributes<HTMLLIElement> & {
  ref?: React.RefObject<HTMLLIElement>;
}) => (
  <li
    ref={ref}
    className={cn("relative flex flex-col p-6 pt-0 *:mb-3", className)}
    {...props}
  />
);
TimelineItem.displayName = "TimelineItem";

const TimelineTime = ({
  ref,
  className,
  ...props
}: React.HTMLAttributes<HTMLParagraphElement> & {
  ref?: React.RefObject<HTMLParagraphElement>;
}) => (
  <p
    ref={ref}
    className={cn(
      "text-secondary-foreground absolute translate-x-36 text-sm leading-none font-semibold md:-translate-x-20",
      className,
    )}
    {...props}
  />
);
TimelineTime.displayName = "TimelineTime";

const TimelineConnector = ({
  ref,
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement> & {
  ref?: React.RefObject<HTMLDivElement>;
}) => (
  <div
    ref={ref}
    className={cn(
      "bg-primary absolute top-[5px] left-[30px] h-full w-px -translate-x-1/2 translate-y-2",
      className,
    )}
    {...props}
  />
);
TimelineConnector.displayName = "TimelineConnector";

const TimelineHeader = ({
  ref,
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement> & {
  ref?: React.RefObject<HTMLDivElement>;
}) => (
  <div
    ref={ref}
    className={cn("flex items-center gap-4", className)}
    {...props}
  />
);
TimelineHeader.displayName = "TimelineHeader";

const TimelineTitle = ({
  ref,
  className,
  children,
  ...props
}: React.HTMLAttributes<HTMLHeadingElement> & {
  ref?: React.RefObject<HTMLHeadingElement>;
}) => (
  <h3
    ref={ref}
    className={cn(
      "text-secondary-foreground leading-none font-semibold tracking-tight",
      className,
    )}
    {...props}
  >
    {children}
  </h3>
);
TimelineTitle.displayName = "CardTitle";

const TimelineIcon = ({
  ref,
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement> & {
  ref?: React.RefObject<HTMLDivElement>;
}) => (
  <div
    ref={ref}
    className={cn("bg-primary flex size-3 flex-col rounded-full", className)}
    {...props}
  />
);
TimelineIcon.displayName = "TimelineIcon";

const TimelineDescription = ({
  ref,
  className,
  ...props
}: React.HTMLAttributes<HTMLParagraphElement> & {
  ref?: React.RefObject<HTMLParagraphElement>;
}) => (
  <p
    ref={ref}
    className={cn("text-muted-foreground text-sm", className)}
    {...props}
  />
);
TimelineDescription.displayName = "TimelineDescription";

const TimelineContent = ({
  ref,
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement> & {
  ref?: React.RefObject<HTMLDivElement>;
}) => (
  <div
    ref={ref}
    className={cn("flex flex-col items-start p-6 pt-0", className)}
    {...props}
  />
);
TimelineContent.displayName = "TimelineContent";

export {
  Timeline,
  TimelineItem,
  TimelineConnector,
  TimelineHeader,
  TimelineTitle,
  TimelineIcon,
  TimelineDescription,
  TimelineContent,
  TimelineTime,
};

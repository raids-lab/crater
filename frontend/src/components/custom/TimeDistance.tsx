// i18n-processed-v1.1.0 (no translatable strings)
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { format, formatDistanceToNow } from "date-fns";
import { zhCN, ja, ko, enUS, type Locale } from "date-fns/locale";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";

export const TimeDistance = ({
  date,
  className,
}: {
  date?: string;
  className?: string;
}) => {
  const [timeDiff, setTimeDiff] = useState("");
  const [startTime, setStartTime] = useState<Date | null>(null);
  const { i18n } = useTranslation();
  const [locale, setLocale] = useState<Locale>(zhCN);

  useEffect(() => {
    switch (i18n.language) {
      case "zh":
        setLocale(zhCN);
        break;
      case "ja":
        setLocale(ja);
        break;
      case "ko":
        setLocale(ko);
        break;
      default:
        setLocale(enUS);
    }
  }, [i18n.language]);

  useEffect(() => {
    if (!date) {
      setStartTime(null);
      setTimeDiff("");
      return;
    }

    const time = new Date(date);
    setStartTime(time);

    const updateTimeDiff = () => {
      const timeDifference = formatDistanceToNow(time, {
        locale,
        addSuffix: true,
      });
      setTimeDiff(timeDifference.replace(/^[^\d]*\s*/, ""));
    };

    updateTimeDiff();
    const timer = setInterval(updateTimeDiff, 10000);
    return () => clearInterval(timer);
  }, [date, locale]);

  if (!startTime) {
    return null;
  }

  return (
    <TooltipProvider delayDuration={10}>
      <Tooltip>
        <TooltipTrigger className={cn("cursor-help", className)}>
          {timeDiff}
        </TooltipTrigger>
        <TooltipContent>
          {format(startTime, "PPPp", { locale: locale })}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
};

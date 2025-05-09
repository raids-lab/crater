// i18n-processed-v1.1.0
// Modified code
import React from "react";
import { useTranslation } from "react-i18next";
import { motion, useMotionValue } from "framer-motion";
import { DropdownMenuLabel } from "@/components/ui/dropdown-menu";
import { Save, LogsIcon, InfoIcon } from "lucide-react";
import CraterIcon from "@/components/icon/CraterIcon";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { Button } from "@/components/ui/button";

export default function FloatingBall({
  handleShowLog,
  handleShowDetail,
  handleSnapshot,
  setIsDragging,
}: {
  handleShowLog: () => void;
  handleSnapshot: () => void;
  handleShowDetail: () => void;
  setIsDragging: (isDragging: boolean) => void;
}) {
  const { t } = useTranslation();
  const x = useMotionValue(window.innerWidth - 60);
  const y = useMotionValue(window.innerHeight - 60);

  const constraintsRef = React.useRef(null);

  return (
    <motion.div
      ref={constraintsRef}
      className="pointer-events-none fixed inset-0"
    >
      <motion.div
        drag
        dragConstraints={constraintsRef}
        dragElastic={0.1}
        dragMomentum={false}
        style={{ x, y }}
        className="pointer-events-auto relative flex h-12 w-12 cursor-move items-center justify-center rounded-full"
        whileHover={{ scale: 1.1 }}
        onDragStart={() => setIsDragging(true)}
        onDragEnd={() => setIsDragging(false)}
      >
        <TooltipProvider delayDuration={200}>
          <Tooltip>
            <TooltipTrigger asChild>
              <CraterIcon
                style={{ filter: "drop-shadow(1px 1px 4px rgba(0,0,0,0.5))" }}
                className="h-12 w-12"
              />
            </TooltipTrigger>
            <TooltipContent
              side="left"
              align="end"
              className="bg-background text-foreground flex w-32 flex-col border p-1 [&_span]:hidden"
            >
              <DropdownMenuLabel className="text-muted-foreground text-xs">
                {t("floatingBall.tooltip.operations")}
              </DropdownMenuLabel>
              {/* 按钮 */}
              <Button
                variant="ghost"
                className="justify-start px-2 py-1 font-normal"
                onClick={handleShowDetail}
              >
                <InfoIcon className="text-primary" />
                {t("floatingBall.tooltip.jobDetails")}
              </Button>
              <Button
                variant="ghost"
                className="justify-start px-2 py-1 font-normal"
                onClick={handleShowLog}
              >
                <LogsIcon className="text-highlight-orange" />
                {t("floatingBall.tooltip.logDiagnosis")}
              </Button>
              <Button
                variant="ghost"
                className="justify-start px-2 py-1 font-normal"
                onClick={handleSnapshot}
              >
                <Save className="text-highlight-purple" />
                {t("floatingBall.tooltip.saveSnapshot")}
              </Button>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </motion.div>
    </motion.div>
  );
}

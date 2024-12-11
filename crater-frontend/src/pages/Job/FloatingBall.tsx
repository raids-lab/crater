import React from "react";
import { motion, useMotionValue } from "framer-motion";
import { DropdownMenuLabel } from "@/components/ui/dropdown-menu";
import { Save, GaugeIcon, LogsIcon, InfoIcon } from "lucide-react";
import { toast } from "sonner";
import CraterIcon from "@/components/icon/CraterIcon";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { Button } from "@/components/ui/button";
import { useNavigate } from "react-router-dom";

export default function FloatingBall({
  jobName,
  handleShowLog,
  setIsDragging,
}: {
  jobName: string;
  handleShowLog: () => void;
  setIsDragging: (isDragging: boolean) => void;
}) {
  const navigate = useNavigate();

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
              />
            </TooltipTrigger>
            <TooltipContent
              side="left"
              align="end"
              className="flex w-32 flex-col border bg-background p-1 text-foreground"
            >
              <DropdownMenuLabel className="text-xs text-muted-foreground">
                操作
              </DropdownMenuLabel>
              {/* 按钮 */}
              <Button
                variant="ghost"
                className="justify-start px-2 py-1"
                onClick={() => navigate(`/portal/job/inter/${jobName}`)}
              >
                <InfoIcon className="text-primary" />
                <span>作业详情</span>
              </Button>
              <Button
                variant="ghost"
                className="justify-start px-2 py-1"
                onClick={() =>
                  window.open(
                    `${import.meta.env.VITE_GRAFANA_JOB_MONITOR}?orgId=1&var-job=${jobName}&from=now-1h&to=now`,
                    "_blank",
                  )
                }
              >
                <GaugeIcon className="text-green-600 dark:text-green-500" />
                资源监控
              </Button>
              <Button
                variant="ghost"
                className="justify-start px-2 py-1"
                onClick={handleShowLog}
              >
                <LogsIcon className="text-orange-600 dark:text-orange-500" />
                日志诊断
              </Button>
              <Button
                variant="ghost"
                className="justify-start px-2 py-1"
                onClick={() =>
                  toast.warning("TODO(liyilong): 保存功能开发中，预计12月上线")
                }
              >
                <Save className="text-purple-600 dark:text-purple-500" />
                <span>保存镜像</span>
              </Button>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </motion.div>
    </motion.div>
  );
}

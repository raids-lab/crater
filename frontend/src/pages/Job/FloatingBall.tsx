"use client";

import React, { useState } from "react";
import { motion, useMotionValue } from "framer-motion";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Save, FileText, MoreVertical } from "lucide-react";
import { toast } from "sonner";
import CraterIcon from "@/components/icon/CraterIcon";

export default function FloatingBall({
  handleShowLog,
}: {
  handleShowLog: () => void;
}) {
  const [isDragging, setIsDragging] = useState(false);
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
        whileTap={{ scale: 0.9 }}
        onDragStart={() => setIsDragging(true)}
        onDragEnd={() => setIsDragging(false)}
      >
        <CraterIcon
          style={{ filter: "drop-shadow(1px 1px 4px rgba(0,0,0,0.5))" }}
        />
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button
              className="absolute left-0 top-0 flex h-7 w-7 -translate-x-1/3 -translate-y-1/3 transform items-center justify-center rounded-full bg-secondary text-secondary-foreground shadow-sm"
              onClick={(e) => {
                if (isDragging) {
                  e.preventDefault();
                  e.stopPropagation();
                }
              }}
            >
              <MoreVertical className="size-4" />
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent side="left">
            <DropdownMenuItem onClick={handleShowLog}>
              <FileText className="mr-2 size-4" />
              <span>日志</span>
            </DropdownMenuItem>
            <DropdownMenuItem
              onClick={() => toast.warning("环境保存功能开发中")}
            >
              <Save className="mr-2 size-4" />
              <span>保存</span>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </motion.div>
    </motion.div>
  );
}

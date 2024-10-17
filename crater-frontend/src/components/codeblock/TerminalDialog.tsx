import { useEffect, useRef } from "react";
import { Card } from "../ui/card";
import { ContainerInfo } from "@/services/api/tool";
import { PodContainerDialog, PodNamespacedName } from "./PodContainerDialog";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import "@xterm/xterm/css/xterm.css";
import { REFRESH_TOKEN_KEY } from "@/utils/store";

const buildWebSocketUrl = (
  namespace: string,
  podName: string,
  containerName: string,
) => {
  const apiBaseUrl = import.meta.env.VITE_API_BASE_URL;
  const version = import.meta.env.VITE_API_VERSION;
  const protocol = window.location.protocol === "https:" ? "wss" : "ws";
  const token = localStorage.getItem(REFRESH_TOKEN_KEY) || "";
  const url = new URL(apiBaseUrl);
  url.protocol = protocol;
  url.pathname =
    url.pathname +
    `${version}/namespaces/${namespace}/pods/${podName}/containers/${containerName}/terminal`;
  return url.toString() + `?token=${token}`;
};

function TerminalCard({
  namespacedName,
  selectedContainer,
}: {
  namespacedName: PodNamespacedName;
  selectedContainer: ContainerInfo;
}) {
  const terminalRef = useRef<Terminal | null>(null); // 用于保存终端实例
  const xtermRef = useRef<HTMLDivElement>(null); // 用于保存终端的 DOM 元素
  const fitAddonRef = useRef<FitAddon | null>(null); // 用于保存 fitAddon 实例
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    if (!selectedContainer.state.running || !xtermRef.current) {
      return;
    }
    const terminal = new Terminal();
    const fitAddon = new FitAddon();

    terminalRef.current = terminal;
    fitAddonRef.current = fitAddon;

    terminal.loadAddon(fitAddon);
    terminal.open(xtermRef.current);

    // Add custom class to xterm-screen
    const xtermScreen = xtermRef.current.querySelector(".xterm");
    if (xtermScreen) {
      xtermScreen.classList.add("custom-xterm-screen");
    }

    terminal.focus();
    fitAddon.fit(); // 调整终端大小

    // 监听窗口调整，自动调整终端大小
    const handleResize = () => {
      fitAddon.fit();
    };
    window.addEventListener("resize", handleResize);

    const wsUrl = buildWebSocketUrl(
      namespacedName.namespace,
      namespacedName.name,
      selectedContainer.name,
    );

    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onmessage = (event: MessageEvent) => {
      terminal.write(event.data as string);
    };

    ws.onerror = () => {
      terminal.writeln("WebSocket error");
    };

    ws.onclose = () => {
      terminal.writeln("Connection closed.");
    };

    // 将终端的输入发送到 WebSocket
    terminal.onData((data) => {
      ws.send(data);
    });

    return () => {
      window.removeEventListener("resize", handleResize);
      ws.close();
      terminal.dispose();
    };
  }, [namespacedName, selectedContainer]);

  return (
    <Card className="overflow-hidden rounded-md bg-black p-1 dark:border dark:bg-muted/30 md:col-span-2 xl:col-span-3">
      <div ref={xtermRef} className="h-full w-full" />
    </Card>
  );
}

export default function TerminalDialog({
  namespacedName,
  setNamespacedName,
}: {
  namespacedName?: PodNamespacedName;
  setNamespacedName: (namespacedName: PodNamespacedName | undefined) => void;
}) {
  return (
    <PodContainerDialog
      namespacedName={namespacedName}
      setNamespacedName={setNamespacedName}
      ActionComponent={TerminalCard}
    />
  );
}

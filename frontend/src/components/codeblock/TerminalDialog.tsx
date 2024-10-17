import { useEffect, useRef } from "react";
import useResizeObserver from "use-resize-observer";
import { Card } from "../ui/card";
import { ContainerInfo } from "@/services/api/tool";
import { PodContainerDialog, PodNamespacedName } from "./PodContainerDialog";
import { Terminal } from "xterm";
import "xterm/css/xterm.css";
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
  const terminalRef = useRef<HTMLDivElement>(null);
  const xtermRef = useRef<Terminal | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const { ref: refRoot, width, height } = useResizeObserver();

  useEffect(() => {
    if (!selectedContainer.state.running || !terminalRef.current) {
      return;
    }
    const terminal = new Terminal({});
    terminal.open(terminalRef.current);
    xtermRef.current = terminal;

    const wsUrl = buildWebSocketUrl(
      namespacedName.namespace,
      namespacedName.name,
      selectedContainer.name,
    );

    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      terminal.writeln(
        `Connected to terminal of ${namespacedName.namespace}/${namespacedName.name}/${selectedContainer.name}...`,
      );
    };

    ws.onmessage = (event: MessageEvent) => {
      terminal.write(event.data as string);
    };

    ws.onerror = () => {
      terminal.writeln("WebSocket error");
    };

    ws.onclose = () => {
      terminal.writeln("Connection closed.");
    };

    terminal.onData((data) => {
      ws.send(data);
    });

    return () => {
      ws.close();
      terminal.dispose();
    };
  }, [namespacedName, selectedContainer]);

  return (
    <Card
      className="relative h-full overflow-hidden rounded-md bg-slate-900 p-1 text-white dark:border dark:bg-muted/30 md:col-span-2 xl:col-span-3"
      ref={refRoot}
    >
      <div ref={terminalRef} style={{ width: width, height: height }} />
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

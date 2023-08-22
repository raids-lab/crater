import { useState } from "react";
import reactLogo from "@/assets/react.svg";
import clsx from "clsx";

function App() {
  const [count, setCount] = useState(0);

  return (
    <>
      <div className="flex flex-col items-center justify-center space-y-4 p-20">
        <a
          href="https://gitlab.act.buaa.edu.cn/gpu-portal"
          target="_blank"
          rel="noreferrer"
          className={clsx("h-40 w-full")}
        >
          <img
            src={reactLogo}
            className="flex h-40 w-full items-center justify-center"
            alt="Vite logo"
          />
        </a>
        <h1 className="text-5xl font-bold">GPU Portal</h1>
        <div className="text-lg">
          <button onClick={() => setCount((count) => count + 1)}>
            count is {count}
          </button>
        </div>
      </div>
    </>
  );
}

export default App;

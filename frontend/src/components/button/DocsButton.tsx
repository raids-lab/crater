import { BookOpenIcon } from "lucide-react";
import { Button } from "../ui/button";

const DocsButton = ({ title, url }: { title: string; url: string }) => {
  return (
    <Button
      variant="secondary"
      onClick={() =>
        window.open(`https://${import.meta.env.VITE_HOST}/website/docs/${url}`)
      }
    >
      <BookOpenIcon />
      {title}
    </Button>
  );
};

export default DocsButton;

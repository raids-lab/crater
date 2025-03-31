import { BookOpenIcon } from "lucide-react";
import { Button, buttonVariants } from "../ui/button";
import { urlWebsiteBaseAtom } from "@/utils/store/config";
import { useAtomValue } from "jotai";
import { VariantProps } from "class-variance-authority";
import { cn } from "@/lib/utils";

type DocsButtonProps = React.ComponentProps<"button"> &
  VariantProps<typeof buttonVariants> & {
    title: string;
    url: string;
  };

const DocsButton = ({
  title,
  url,
  variant,
  className,
  ...props
}: DocsButtonProps) => {
  const website = useAtomValue(urlWebsiteBaseAtom);
  return (
    <Button
      variant={variant ?? "secondary"}
      className={cn("cursor-pointer", className)}
      onClick={() => window.open(`${website}/docs/${url}`)}
      {...props}
    >
      <BookOpenIcon />
      {title}
    </Button>
  );
};

export default DocsButton;

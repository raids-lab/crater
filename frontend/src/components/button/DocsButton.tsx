import { BookOpenIcon } from "lucide-react";
import { Button, buttonVariants } from "../ui/button";
import { configUrlWebsiteBaseAtom } from "@/utils/store/config";
import { useAtomValue } from "jotai";
import { VariantProps } from "class-variance-authority";
import { cn } from "@/lib/utils";
import { Link } from "react-router-dom";

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
  const website = useAtomValue(configUrlWebsiteBaseAtom);
  return (
    <Button
      variant={variant ?? "secondary"}
      className={cn("cursor-pointer", className)}
      {...props}
      asChild
    >
      <Link to={`${website}/docs/user/${url}`} reloadDocument>
        <BookOpenIcon />
        {title}
      </Link>
    </Button>
  );
};

export default DocsButton;

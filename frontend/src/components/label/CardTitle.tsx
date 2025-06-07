import { LucideIcon } from "lucide-react";

type CardTitleProps = React.HTMLAttributes<HTMLDivElement> & {
  icon: LucideIcon;
};

const CardTitle = ({ children, ...props }: CardTitleProps) => {
  return (
    <div className="flex items-center gap-1.5 text-base font-semibold">
      <props.icon className="text-primary size-5" />
      {children}
    </div>
  );
};

export default CardTitle;

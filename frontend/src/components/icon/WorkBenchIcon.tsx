import { SVGProps, Ref, forwardRef } from "react";
const SvgComponent = (
  props: SVGProps<SVGSVGElement>,
  ref: Ref<SVGSVGElement>,
) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    width={15}
    height={15}
    fill="none"
    viewBox="0 0 48 48"
    ref={ref}
    {...props}
  >
    <path stroke="currentColor" strokeWidth={3} d="M12 33H4V7h40v26H12Z" />
    <path
      stroke="currentColor"
      strokeLinecap="square"
      strokeWidth={3}
      d="M16 22v4M24 33v6M24 18v8M32 14v12M12 41h24"
    />
  </svg>
);
const ForwardRef = forwardRef(SvgComponent);
export default ForwardRef;

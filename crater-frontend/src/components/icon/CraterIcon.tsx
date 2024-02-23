import { SVGProps, Ref, forwardRef } from "react";
const SvgComponent = (
  props: SVGProps<SVGSVGElement>,
  ref: Ref<SVGSVGElement>,
) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    xmlSpace="preserve"
    viewBox="0 0 200 200"
    ref={ref}
    {...props}
  >
    <title>{"Crater"}</title>
    <path
      fill="currentColor"
      d="M185.84 99.99h-.01l.01.01v49.56l-42.92 24.78L100 199.12l-85.84-49.56.32-.18-.32-.56V50.44L100 .88l42.92 24.78v49.55l42.91-24.77h.01z"
    />
    <path
      fill="#DE553E"
      d="M142.92 25.66v99.11L100 149.56l-42.92-24.78V75.21L100 50.44z"
    />
    <path
      fill="#FAAC3E"
      d="M142.92 75.21v49.56L100 149.56V99.99l42.91-24.78z"
    />
  </svg>
);
const ForwardRef = forwardRef(SvgComponent);
export default ForwardRef;

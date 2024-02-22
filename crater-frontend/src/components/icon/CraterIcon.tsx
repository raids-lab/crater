import { SVGProps, Ref, forwardRef } from "react";
const SvgComponent = (
  props: SVGProps<SVGSVGElement>,
  ref: Ref<SVGSVGElement>,
) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    xmlSpace="preserve"
    viewBox="0 0 560 560"
    ref={ref}
    {...props}
  >
    <title>{"Layer 1"}</title>
    <g stroke="null">
      <path
        fill="currentColor"
        d="M515.015 280.012v135.676l-117.51-67.838v-.035l117.475-67.838zM515.015 144.312v135.665h-.035l-117.475-67.804v-.034l117.475-67.827z"
      />
      <path
        fill="currentColor"
        d="m397.505 347.85 117.51 67.838-117.51 67.838zM514.98 279.977l-117.475 67.838V212.173zM397.505 347.85v135.676l-117.5-67.838 117.5-67.873z"
      />
      <path
        fill="#FFC25F"
        d="M397.505 212.173v135.642l-117.5-67.838 117.477-67.838h.023z"
        className="st0"
      />
      <path
        fill="#F7931E"
        d="M397.505 76.474v135.665h-.023l-117.476-67.827z"
        className="st1"
      />
      <path fill="currentColor" d="m397.505 76.474-117.5 67.838V8.635z" />
      <path
        fill="#FFC25F"
        d="m397.505 347.815-117.5 67.873V279.977z"
        className="st0"
      />
      <path fill="currentColor" d="m397.505 483.526-117.5 67.839V415.688z" />
      <path
        fill="#F7931E"
        d="m397.482 212.139-117.476 67.838V144.312z"
        className="st1"
      />
      <path fill="currentColor" d="M280.006 415.688v135.677l-117.51-67.839z" />
      <path
        fill="#F7931E"
        d="M280.006 279.977v135.711l-117.51-67.838v-.035zM280.006 144.312v135.665l-117.51-67.838z"
        className="st1"
      />
      <path
        fill="currentColor"
        d="M280.006 8.635v135.677l-117.51-67.838zM280.006 144.312l-117.51 67.827V76.474z"
      />
      <path
        fill="#F7931E"
        d="m280.006 279.977-117.51 67.838V212.14z"
        className="st1"
      />
      <path
        fill="currentColor"
        d="m280.006 415.688-117.51 67.838V347.85zM162.495 347.85v135.676l-117.51-67.838.887-.507zM162.495 347.815v.035l-117.51-67.838 117.51-67.873zM162.495 76.474v135.665l-117.51-67.827z"
      />
      <path
        fill="currentColor"
        d="m162.495 212.139-117.51 67.873v-135.7zM162.495 347.85 45.872 415.18l-.887-1.532V280.012z"
      />
    </g>
  </svg>
);
const ForwardRef = forwardRef(SvgComponent);
export default ForwardRef;

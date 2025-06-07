import type { IconProps } from "../../../types/icons";

const ClockDark = (props: IconProps) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    preserveAspectRatio="xMidYMid meet"
    viewBox="0 0 425 440"
    fill="none"
    {...props}
  >
    <g filter="url(#a)">
      <circle cx={207.516} cy={190.516} r={180.516} fill="#606060" />
    </g>
    <g filter="url(#b)">
      <circle cx={207.516} cy={190.516} r={174.215} fill="#272727" />
    </g>
    <path
      fill="#C9C9C9"
      d="m61.902 111.625-.891 1.544 49.408 28.526.892-1.544zM110.418 238.652l.892 1.544L61.9 268.722l-.891-1.544zM292.203 51.121l1.544.892-28.526 49.408-1.544-.892zM121.047 52.012l1.544-.892 28.526 49.409-1.544.891zM206.625 26.492h1.783v57.052h-1.783zM355.121 111.625l.892 1.544-49.409 28.526-.891-1.544zM306.605 238.652l-.892 1.544 49.409 28.526.891-1.544zM320.285 194.082v-1.783h57.052v1.783zM45.902 193.98v-1.89h56.706v1.89zM122.586 332.75l-1.544-.892 28.526-49.408 1.544.892zM293.742 331.855l-1.544.892-28.526-49.409 1.544-.891zM206.379 301.059h1.783v57.052h-1.783z"
    />
    <g filter="url(#c)">
      <circle cx={207.516} cy={190.516} r={174.215} fill="#9C99A6" fillOpacity={0.1} />
    </g>
    <defs>
      <filter
        id="a"
        width={423.631}
        height={438.631}
        x={0.7}
        y={0.7}
        colorInterpolationFilters="sRGB"
        filterUnits="userSpaceOnUse"
      >
        <feFlood floodOpacity={0} result="BackgroundImageFix" />
        <feColorMatrix in="SourceAlpha" result="hardAlpha" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0" />
        <feMorphology in="SourceAlpha" radius={1} result="effect1_dropShadow_50_62" />
        <feOffset dx={5} dy={37} />
        <feGaussianBlur stdDeviation={16.15} />
        <feComposite in2="hardAlpha" operator="out" />
        <feColorMatrix values="0 0 0 0 0.036884 0 0 0 0 0.036884 0 0 0 0 0.036884 0 0 0 0.8 0" />
        <feBlend in2="BackgroundImageFix" result="effect1_dropShadow_50_62" />
        <feBlend in="SourceGraphic" in2="effect1_dropShadow_50_62" result="shape" />
        <feTurbulence baseFrequency="1 1" numOctaves={3} seed={2256} type="fractalNoise" />
        <feDisplacementMap
          width="100%"
          height="100%"
          in="shape"
          result="displacedImage"
          scale={18.6}
          xChannelSelector="R"
          yChannelSelector="G"
        />
        <feMerge result="effect2_texture_50_62">
          <feMergeNode in="displacedImage" />
        </feMerge>
      </filter>
      <filter
        id="b"
        width={348.43}
        height={464.63}
        x={33.301}
        y={-26.899}
        colorInterpolationFilters="sRGB"
        filterUnits="userSpaceOnUse"
      >
        <feFlood floodOpacity={0} result="BackgroundImageFix" />
        <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
        <feColorMatrix in="SourceAlpha" result="hardAlpha" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0" />
        <feMorphology in="SourceAlpha" radius={24} result="effect1_innerShadow_50_62" />
        <feOffset dy={73} />
        <feGaussianBlur stdDeviation={28.5} />
        <feComposite in2="hardAlpha" k2={-1} k3={1} operator="arithmetic" />
        <feColorMatrix values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0.54 0" />
        <feBlend in2="shape" result="effect1_innerShadow_50_62" />
        <feColorMatrix in="SourceAlpha" result="hardAlpha" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0" />
        <feOffset dy={-49} />
        <feGaussianBlur stdDeviation={21.6} />
        <feComposite in2="hardAlpha" k2={-1} k3={1} operator="arithmetic" />
        <feColorMatrix values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0.33 0" />
        <feBlend in2="effect1_innerShadow_50_62" result="effect2_innerShadow_50_62" />
      </filter>
      <filter
        id="c"
        width={348.43}
        height={348.43}
        x={33.301}
        y={16.301}
        colorInterpolationFilters="sRGB"
        filterUnits="userSpaceOnUse"
      >
        <feFlood floodOpacity={0} result="BackgroundImageFix" />
        <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
        <feTurbulence
          baseFrequency="0.08474576473236084 0.08474576473236084"
          numOctaves={3}
          result="noise"
          seed={8854}
          stitchTiles="stitch"
          type="fractalNoise"
        />
        <feColorMatrix in="noise" result="alphaNoise" type="luminanceToAlpha" />
        <feComponentTransfer in="alphaNoise" result="coloredNoise1">
          <feFuncA
            tableValues="0 0 0 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0"
            type="discrete"
          />
        </feComponentTransfer>
        <feComposite in="coloredNoise1" in2="shape" operator="in" result="noise1Clipped" />
        <feFlood floodColor="rgba(0, 0, 0, 0.96)" result="color1Flood" />
        <feComposite in="color1Flood" in2="noise1Clipped" operator="in" result="color1" />
        <feMerge result="effect1_noise_50_62">
          <feMergeNode in="shape" />
          <feMergeNode in="color1" />
        </feMerge>
      </filter>
    </defs>
  </svg>
);
export default ClockDark;

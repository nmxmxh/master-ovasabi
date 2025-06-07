import type { IconProps } from "../../../types/icons";

const ClockLight = (props: IconProps) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    preserveAspectRatio="xMidYMid meet"
    viewBox="0 0 425 440"
    fill="none"
    {...props}
  >
    <g filter="url(#a)">
      <circle cx={207.5} cy={190.5} r={180.5} fill="#E9E5E0" />
    </g>
    <g filter="url(#b)">
      <circle cx={207.499} cy={190.499} r={174.26} fill="#DBDADF" />
    </g>
    <path
      fill="#595252"
      d="m61.898 111.613-.891 1.544 49.404 28.524.892-1.544zM110.41 238.633l.891 1.544-49.404 28.524-.891-1.544zM292.18 51.117l1.544.892-28.523 49.404-1.544-.892zM121.04 52.008l1.543-.892 28.524 49.405-1.544.891zM206.609 26.488h1.783v57.047h-1.783zM355.09 111.613l.892 1.544-49.405 28.524-.891-1.544zM306.578 238.633l-.892 1.544 49.405 28.524.891-1.545zM320.258 194.066v-1.783h57.047v1.783zM45.719 194.066v-1.783h57.047v1.783zM122.582 332.723l-1.544-.892 28.523-49.404 1.544.892zM293.723 331.828l-1.544.892-28.524-49.405 1.545-.891zM206.371 301.031h1.783v57.047h-1.783z"
    />
    <g filter="url(#c)">
      <circle cx={207.499} cy={190.499} r={174.26} fill="#9C99A6" fillOpacity={0.1} />
    </g>
    <defs>
      <filter
        id="a"
        width={423.6}
        height={438.6}
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
        <feColorMatrix values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0.31 0" />
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
        width={348.52}
        height={412.92}
        x={33.238}
        y={16.238}
        colorInterpolationFilters="sRGB"
        filterUnits="userSpaceOnUse"
      >
        <feFlood floodOpacity={0} result="BackgroundImageFix" />
        <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
        <feColorMatrix in="SourceAlpha" result="hardAlpha" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0" />
        <feMorphology in="SourceAlpha" radius={15} result="effect1_innerShadow_50_62" />
        <feOffset dy={69} />
        <feGaussianBlur stdDeviation={24.7} />
        <feComposite in2="hardAlpha" k2={-1} k3={1} operator="arithmetic" />
        <feColorMatrix values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0.25 0" />
        <feBlend in2="shape" result="effect1_innerShadow_50_62" />
      </filter>
      <filter
        id="c"
        width={348.52}
        height={348.52}
        x={33.238}
        y={16.238}
        colorInterpolationFilters="sRGB"
        filterUnits="userSpaceOnUse"
      >
        <feFlood floodOpacity={0} result="BackgroundImageFix" />
        <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
        <feTurbulence
          baseFrequency="0.087719298899173737 0.087719298899173737"
          numOctaves={3}
          result="noise"
          seed={8854}
          stitchTiles="stitch"
          type="fractalNoise"
        />
        <feColorMatrix in="noise" result="alphaNoise" type="luminanceToAlpha" />
        <feComponentTransfer in="alphaNoise" result="coloredNoise1">
          <feFuncA
            tableValues="1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0"
            type="discrete"
          />
        </feComponentTransfer>
        <feComposite in="coloredNoise1" in2="shape" operator="in" result="noise1Clipped" />
        <feFlood floodColor="rgba(0, 0, 0, 0.4)" result="color1Flood" />
        <feComposite in="color1Flood" in2="noise1Clipped" operator="in" result="color1" />
        <feMerge result="effect1_noise_50_62">
          <feMergeNode in="shape" />
          <feMergeNode in="color1" />
        </feMerge>
      </filter>
    </defs>
  </svg>
);

export default ClockLight;

import type { IconProps } from "../../types/icons";

const Logo = (props: IconProps) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    preserveAspectRatio="xMidYMid meet"
    viewBox="0 0 50 49"
    fill="none"
    {...props}
  >
    <path className="logo" fill="#121212" d="M0 0h5.808v20.229H0V0Z" />
    <path
      className="logo"
      fill="#121212"
      d="M2.392 5.829V0h17.766v5.829H2.392ZM4.783 20.229V14.4h15.375v5.829H4.783ZM28.7 5.829V0h20.157v5.829H28.7ZM21.183 20.229V14.4h12.983v2.4l-2.904 3.429h-10.08ZM35.19 16.8v-5.829h7.859V16.8h-7.858Z"
    />
    <path
      className="logo"
      fill="#121212"
      d="M14.35 0h5.808v17.829H14.35V0ZM0 28.8v-5.829h14.35V28.8H0ZM0 39.771v-5.828h20.158v5.828H0Z"
    />
    <path
      className="logo"
      fill="#121212"
      d="M14.35 33.943h5.808V48H14.35V33.943ZM15.375 22.971h5.466v9.943h-5.466v-9.943Z"
    />
    <path
      className="logo"
      fill="#121212"
      d="M20.158 48v-5.829H0V48h20.158ZM15.375 28.8v-5.829h11.958V28.8H15.375ZM21.524 39.771v-5.828H28.7v5.828h-7.175ZM0 22.971h5.808v12H0v-12ZM48.857 42.171V48H35.191v-5.829h13.666ZM48.857 31.543v5.828H35.191v-5.828h13.666Z"
    />
    <path
      className="logo"
      fill="#121212"
      d="M48.857 48H43.05V36.343h5.808V48ZM34.166 48h-6.833V22.971h6.833V48ZM34.166 16.8h-6.833V0h6.833v16.8ZM43.049 22.971h5.808v6.515H43.05V22.97ZM43.049 2.743h5.808v17.486H43.05V2.743Z"
    />
  </svg>
);
export default Logo;

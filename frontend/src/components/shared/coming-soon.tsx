import styled from "styled-components";
import Logo from "../icons/logo";
import ClockDark from "../icons/clock-dark";
import ClockHandsDiv from "../icons/clock-dark/hands";
import { useUIStore } from "../../store/global";
import ClockLight from "../icons/clock-light";

export function ComingSoon() {
  const theme = useUIStore((state) => state.theme);
  return (
    <Style.Container>
      <figure className="logo">
        <Logo />
      </figure>
      <figure className="clock">
        {theme === "dark" ? <ClockDark className="clock-face" /> : <ClockLight />}
        <ClockHandsDiv $theme={theme} className="clock-hands" />
      </figure>
      <hgroup>
        <h1>Something Bold Is Brewing at Ovasabi!</h1>
        <h2>
          We're reimagining the modern digital experience.
          <br /> Our new website is almost ready, and it's worth the wait.
        </h2>
      </hgroup>
    </Style.Container>
  );
}

const Style = {
  Container: styled.div`
    height: 100dvh;
    width: 100dvw;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    position: fixed;
    z-index: 50;
    background: #f2e9da;
    text-align: center;
    top: 0;
    overflow: hidden;

    figure.clock {
      display: flex;
      align-items: center;
      justify-content: center;
      height: 55dvh;
      aspect-ratio: 1 / 1;
      position: relative;
      margin-top: 2.5%;

      .clock-face {
        z-index: 1;
      }

      .clock-hands {
        z-index: 2;
      }

      svg {
        position: absolute;
        height: 100%;
      }
    }

    figure.logo {
      height: 7.5%;

      svg {
        height: 100%;
      }
    }

    hgroup {
      display: flex;
      flex-direction: column;
      align-items: center;

      h1 {
        font-family: "geist";
        font-weight: 600;
        font-size: 4.5dvh;
        line-height: 120%;
        letter-spacing: -0.11rem;
        text-align: center;
        vertical-align: middle;
        color: #333234;
        margin-bottom: 0.75dvh;
      }

      h2 {
        font-family: "gordita";
        font-weight: 500;
        font-size: 18px;
        line-height: 150%;
        letter-spacing: -0.05rem;
        text-align: center;
      }
    }
  `,
};

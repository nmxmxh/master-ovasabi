import { useEffect, useRef } from "react";
import { gsap } from "gsap";
import styled from "styled-components";
import { useSound } from "../../../lib/hooks/useSound";

const ClockHandsDiv = (props: any) => {
  const hourRef = useRef<HTMLDivElement>(null);
  const minuteRef = useRef<HTMLDivElement>(null);
  const secondRef = useRef<HTMLDivElement>(null);
  const { $theme } = props;
  const { mute, load, play, stop } = useSound();

  // Track previous rotations to handle 360° transitions
  const prevRotations = useRef({ hour: 0, minute: 0, second: 0 });

  useEffect(() => {
    window.addEventListener("blur", () => mute(true));
    return () => window.addEventListener("focus", () => mute(false));
  }, [mute]);

  useEffect(() => {
    load("second_tick", { loop: true });
    play("second_tick");
    return () => {
      stop("second_tick");
    };
  }, []);

  useEffect(() => {
    const updateClock = () => {
      const now = new Date();
      const seconds = now.getSeconds();
      const minutes = now.getMinutes();
      const hours = (now.getHours() % 12) + minutes / 60;

      const hourDeg = (hours / 12) * 360;
      const minuteDeg = (minutes / 60) * 360;
      const secondDeg = (seconds / 60) * 360;

      // Handle 360° to 0° transitions smoothly
      const getSmoothedRotation = (currentDeg: number, prevDeg: number) => {
        const diff = currentDeg - prevDeg;
        if (diff < -180) {
          return prevDeg + (diff + 360);
        } else if (diff > 180) {
          return prevDeg + (diff - 360);
        }
        return currentDeg;
      };

      const smoothHourDeg = getSmoothedRotation(hourDeg, prevRotations.current.hour);
      const smoothMinuteDeg = getSmoothedRotation(minuteDeg, prevRotations.current.minute);
      const smoothSecondDeg = getSmoothedRotation(secondDeg, prevRotations.current.second);

      gsap.to(hourRef.current, {
        rotate: smoothHourDeg,
        duration: 1.2,
        ease: "elastic.out(1, 0.4)",
      });

      gsap.to(minuteRef.current, {
        rotate: smoothMinuteDeg,
        duration: 1.2,
        ease: "elastic.out(1, 0.4)",
      });

      gsap.to(secondRef.current, {
        rotate: smoothSecondDeg,
        duration: 0.8,
        ease: "elastic.out(1, 0.5)",
      });

      // Update previous rotations
      prevRotations.current = {
        hour: smoothHourDeg,
        minute: smoothMinuteDeg,
        second: smoothSecondDeg,
      };
    };

    updateClock();
    const interval = setInterval(updateClock, 1000);
    return () => clearInterval(interval);
  }, []);

  return (
    <Style.Container {...props}>
      <div ref={hourRef} className="hour" />
      <div ref={minuteRef} className="minute" />
      <div ref={secondRef} className="second" />
      <Style.CenterDot $theme={$theme} />
    </Style.Container>
  );
};

export default ClockHandsDiv;

const Style = {
  Container: styled.div<{ $theme: "light" | "dark" }>`
    position: absolute;
    inset: 0;
    pointer-events: none;
    display: flex;
    align-items: center;
    justify-content: center;

    .hour,
    .minute,
    .second {
      position: absolute;
      transform: translate(-50%, -90%) rotate(0deg);
    }

    .hour {
      height: 12.5dvh;
      width: 1dvh;
      background: ${({ $theme }) => ($theme === "dark" ? "#c9c9c9" : "#1C1C1E")};
      z-index: 1;
      transform-origin: 50% 90%;
      top: 44%;
      left: 49%;
      box-shadow:
        3px 3px 6px rgba(0, 0, 0, 0.3),
        inset -1px 0 2px rgba(0, 0, 0, 0.2),
        inset 1px 0 2px rgba(255, 255, 255, 0.3);
    }

    .minute {
      height: 17.5dvh;
      width: 0.8dvh;
      background: ${({ $theme }) => ($theme === "dark" ? "#c9c9c9" : "#1C1C1E")};
      transition: background 0.25s linear;
      z-index: 2;
      transform-origin: 50% 90%;
      top: 44.5%;
      left: 49%;
      box-shadow:
        3px 3px 6px rgba(0, 0, 0, 0.3),
        inset -1px 0 2px rgba(0, 0, 0, 0.2),
        inset 1px 0 2px rgba(255, 255, 255, 0.3);
    }

    .second {
      height: 20dvh;
      width: 0.35dvh;
      background: #e53935;
      transition: background 0.25s linear;
      z-index: 0;
      transform-origin: 50% 90%;
      top: 44%;
      left: 48.75%;
      box-shadow:
        3px 3px 6px rgba(0, 0, 0, 0.3),
        inset -0.5px 0 1px rgba(0, 0, 0, 0.3),
        inset 0.5px 0 1px rgba(255, 100, 100, 0.4);
    }
  `,
  CenterDot: styled.div<{ $theme: "light" | "dark" }>`
    position: absolute;
    width: 2dvh;
    height: 2dvh;
    background: ${({ $theme }) => ($theme === "dark" ? "#c9c9c9" : "#1C1C1E")};
    border-radius: 50%;
    z-index: 3;
    top: 42.5%;
    left: 47%;
    transition: background 0.25s linear;
    box-shadow:
      2px 2px 6px rgba(0, 0, 0, 0.4),
      inset -1px -1px 2px rgba(0, 0, 0, 0.3),
      inset 1px 1px 3px rgba(255, 255, 255, 0.6);
  `,
};

import { useGSAP } from '@gsap/react';
import gsap from 'gsap';
import SplitText from 'gsap/SplitText';
import _, { sortBy } from 'lodash';
import { useCallback, useRef } from 'react';
import { business_dialogue } from '../../../dialogue/business';
import { type Question, useQuestionnaire } from '../../../lib/hooks/useQuestionnaire';

gsap.registerPlugin(SplitText);
gsap.registerPlugin(useGSAP);

const questionsOnly = _.pickBy(business_dialogue, (_, key) => key.startsWith('question')) as Record<
  string,
  Question
>;

export function useBusinessLayout() {
  const {
    currentQuestion,
    toggleOption,
    selectedOptions,
    next: goNext,
    prev: goPrev,
    currentKey,
    currentIndex,
    totalQuestions
  } = useQuestionnaire(questionsOnly, 'business', undefined);

  const questionRef = useRef<HTMLHeadingElement>(null);
  const numberRef = useRef<HTMLHeadingElement>(null);
  const subtitleRef = useRef<HTMLHeadingElement>(null);
  const captionRef = useRef<HTMLHeadingElement>(null);
  const optionsRef = useRef<HTMLDivElement>(null);

  const nextRef = useRef<HTMLButtonElement>(null);
  const previousRef = useRef<HTMLButtonElement>(null);

  const timelineRef = useRef<gsap.core.Timeline | null>(null);
  const splitRef = useRef<SplitText | null>(null);

  useGSAP(
    () => {
      const el = questionRef.current;
      if (!el || !el.isConnected) return;
      if (!optionsRef.current?.children) return;

      const split = new SplitText(el, {
        type: 'words',
        wordsClass: 'word'
      });
      splitRef.current = split;
      const words = splitRef.current.words;

      const timeline = gsap.timeline({ defaults: { ease: 'power2.out' } });
      timelineRef.current = timeline;

      const additionalEls = Array.from(
        document.querySelectorAll('.light-light, .dark-light, .no-split')
      );
      const filteredSplitWords = words.filter(el => !additionalEls.includes(el));
      const animatedEls = [...filteredSplitWords, ...additionalEls];

      const sortedEls = sortBy(animatedEls, el => {
        return Array.prototype.indexOf.call(el.parentNode?.children, el);
      });

      sortedEls.forEach((el, i) => {
        const randomTilt = gsap.utils.random(-5, 5);
        timeline.fromTo(
          el,
          {
            autoAlpha: 0,
            y: -200,
            rotateZ: randomTilt
          },
          {
            autoAlpha: 1,
            y: 0,
            rotateZ: 0,
            duration: 0.5
          },
          0.4 + i * 0.06
        );
      });

      timeline
        .fromTo(
          numberRef.current,
          { autoAlpha: 0, scale: 0.75 },
          { autoAlpha: 1, scale: 1, duration: 0.25 },
          0.15
        )
        .fromTo(
          previousRef.current,
          { autoAlpha: 0, x: -100 },
          { autoAlpha: 1, x: 0, duration: 0.4 },
          'entry+0.5'
        )
        .fromTo(
          nextRef.current,
          { autoAlpha: 0, x: 100 },
          { autoAlpha: 1, x: 0, duration: 0.4 },
          'entry+0.5'
        )
        .fromTo(captionRef.current, { y: 100, autoAlpha: 0 }, { y: 0, autoAlpha: 1 }, 'entry+=0.45')
        .fromTo(
          optionsRef.current?.children,
          { x: 20, autoAlpha: 0 },
          { x: 0, autoAlpha: 1, stagger: 0.15 },
          'entry+=0.4'
        );

      // Floating and tilt animation after timeline is done
      timeline.add(() => {
        (sortedEls as HTMLElement[]).forEach(el => {
          const floatAnim = gsap.to(el, {
            y: '+=5',
            rotateZ: `+=${gsap.utils.random(-2, 2)}`, // subtle angle tilt
            repeat: -1,
            yoyo: true,
            ease: 'sine.inOut',
            duration: gsap.utils.random(2, 3)
          });

          // Store animation on DOM node so it can be killed later
          (el as any)._floatTilt = floatAnim;
        });
      }, timeline.duration());

      return () => {
        if (splitRef.current) {
          splitRef.current.revert();
          splitRef.current = null;
        }

        if (timelineRef.current) {
          timelineRef.current.kill();
          timelineRef.current = null;
        }

        (sortedEls as HTMLElement[]).forEach((el: any) => {
          if (el._floatTilt) {
            el._floatTilt.kill();
            delete el._floatTilt;
          }
        });
      };
    },
    { dependencies: [currentKey], scope: questionRef }
  );

  const playExitReverse = (callback: () => void) => {
    if (!timelineRef.current) {
      callback();
      return;
    }

    timelineRef.current.timeScale(4);
    timelineRef.current.eventCallback('onReverseComplete', () => {
      callback();
      timelineRef.current?.eventCallback('onReverseComplete', null);
    });
    timelineRef.current.reverse(0);
  };

  const next = useCallback(() => playExitReverse(goNext), [goNext]);
  const prev = useCallback(() => playExitReverse(goPrev), [goPrev]);

  return {
    currentIndex,
    totalQuestions,
    questionRef,
    currentQuestion,
    prev,
    next,
    optionsRef,
    toggleOption,
    currentKey,
    selectedOptions,
    captionRef,
    subtitleRef,
    numberRef,
    nextRef,
    previousRef
  };
}

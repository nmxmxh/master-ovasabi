import { Howler, Howl } from "howler";
import _ from "lodash";

const soundfile_prefix = "/sounds/";

let sounds = [{ name: "second_tick", src: soundfile_prefix + "tick.aac" }];

const defaultSoundOptions = {
  usingWebAudio: true,
  html5: false,
};

function findSound(name: string): { name: string; src: string } | undefined {
  const soundSrc = _.find(sounds, { name: name });

  if (!soundSrc) {
    console.error("Sound not found: " + name);
    return;
  }

  return soundSrc;
}

export function useSound() {
  let loadedSounds: never[] = [];
  function unload() {
    Howler.unload();
    sounds = [];
  }

  function stopAll() {
    Howler.stop();
  }

  function mute(isMuted: boolean) {
    Howler.mute(isMuted);
  }

  function volumeLevel(level: number) {
    Howler.volume(level);
  }

  async function load(name: string, options?: {}) {
    let sound = findSound(name);
    if (!sound) return;

    return new Promise((res, rej) => {
      const howl = new Howl({
        src: sound?.src as any,
        ...defaultSoundOptions,
        ...options,
      });

      (loadedSounds as any)[name] = howl;

      howl.once("load", function () {
        res(howl);
      });
      howl.once("loaderror", function () {
        rej(howl);
      });
    });
  }

  async function loadAll(options: {}) {
    return Promise.all(
      sounds.map(async (sound) => {
        await load(sound.name, options);
      })
    );
  }

  async function play(name: string) {
    let sound: any = null;
    if (typeof (loadedSounds as any)[name] === "undefined") {
      sound = await load(name);
    } else {
      sound = (loadedSounds as any)[name];
    }

    return new Promise((resolve) => {
      sound.once("end", () => {
        resolve(sound);
      });

      sound.once("stop", () => {
        resolve(sound);
      });

      sound.play();
    });
  }

  async function pause(name: string) {
    let sound = (loadedSounds as any)[name];
    if (typeof sound !== "undefined") {
      sound.pause();
    }
  }

  async function stop(name: string) {
    let sound = (loadedSounds as any)[name];
    if (typeof sound !== "undefined") {
      sound.fade(1, 0, 0.75);
      sound.stop();
    }
  }

  return {
    unload,
    stopAll,
    mute,
    volumeLevel,
    load,
    loadAll,
    play,
    pause,
    stop,
  };
}

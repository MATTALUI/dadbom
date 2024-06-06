import { pause, play } from "./icons";
import { getAudioFile } from "./state";
import { formatSeconds } from "./time";

const AUDIO_PLAYER_BUTTON_SELECTOR = "button[aria-label='Audio Player']";
const CLOSE_PLAYER_BUTTON_SELECTOR = "button[aria-label='Close']";
const PLAY_BUTTON_SELECTOR = "button[aria-label='Play']";
const TIME_INPUT_SELECTOR = "input[aria-label='Percent played']";

const waitForEle = async <T extends Element>(
  selector: string,
  intervalMS = 100
): Promise<T> => {
  return new Promise((resolve) => {
    const interval = setInterval(() => {
      const ele = document.querySelector<T>(selector);
      if (ele) {
        clearInterval(interval);
        resolve(ele);
      }
    }, intervalMS);
  });
}

const removePlayButtonListeners = () => {
  const playButton = getPlayButton();
  if (!playButton) return;
  playButton.replaceWith(playButton.cloneNode(true));
}

const addPlayButtonIntercepts = () => {
  const playButton = getPlayButton();
  if (!playButton) return;
  playButton.addEventListener("click", () => {
    const audioFile = getAudioFile();
    if (audioFile.paused) {
      audioFile.play();
      playButton.innerHTML = pause;
    } else {
      audioFile.pause();
      playButton.innerHTML = play;
    }
  });
}

const reflectAudioTimeChanges = (e: Event) => {
  const timeInput = getTimeInput();
  const progressBar = getProgressBar();
  const startTime = getStartTime();
  if (!timeInput || !progressBar || !startTime) return;
  const audioFile = e.target as HTMLAudioElement;
  const time = audioFile.currentTime;
  const duration = audioFile.duration;
  const completionPct = ((time / duration) * 100).toFixed(2);
  timeInput.value = completionPct;
  progressBar.style.width = `${completionPct}%`;
  startTime.innerHTML = formatSeconds(time);
}

export const getPlayButton = (): HTMLDivElement | null => {
  return document.querySelector(PLAY_BUTTON_SELECTOR);
}

export const getTimeInput = (): HTMLInputElement | null => {
  return document.querySelector(TIME_INPUT_SELECTOR);
}

export const getProgressBar = (): HTMLDivElement | null => {
  const timeInput = getTimeInput();
  const progressBar = timeInput?.parentElement?.children[2];
  if (!progressBar) return null;
  return progressBar as HTMLDivElement;
}

export const getStartTime = (): HTMLSpanElement | null => {
  const timeInput = getTimeInput();
  const startTime = timeInput?.parentElement?.parentElement?.children[0];
  if (!startTime) return null;
  return startTime as HTMLSpanElement;
}

export const getEndTime = (): HTMLSpanElement | null => {
  const timeInput = getTimeInput();
  const endTime = timeInput?.parentElement?.parentElement?.children[2];
  if (!endTime) return null;
  return endTime as HTMLSpanElement;
}

export const getAudioPlayerButton = (): HTMLDivElement | null => {
  return document.querySelector(AUDIO_PLAYER_BUTTON_SELECTOR);
}

export const disableAudioPlayerButton = () => {
  getAudioPlayerButton()?.setAttribute("disabled", "true");
}

export const enableAudioPlayerButton = () => {
  getAudioPlayerButton()?.removeAttribute("disabled");
}

export const addAudioPlayerIntercepts = () => {
  const audioFile = getAudioFile();
  const audioPlayerButton = getAudioPlayerButton();
  if (!audioPlayerButton) return;
  audioPlayerButton.addEventListener("click", async () => {
    // We assume that once the close button is visible, everything is
    const closeButton = await waitForEle(CLOSE_PLAYER_BUTTON_SELECTOR);
    // When we close the audio player we will have to reapply the intercepts
    closeButton.addEventListener("click", async () => {
      const audioFile = getAudioFile();
      audioFile.pause();
      audioFile.currentTime = 0;
      await waitForEle(AUDIO_PLAYER_BUTTON_SELECTOR);
      addAudioPlayerIntercepts();
      audioFile.removeEventListener("timeupdate", reflectAudioTimeChanges);
    });
    // Add the intercept for the playbutton
    removePlayButtonListeners();
    addPlayButtonIntercepts();
    // Listen for time changes
    audioFile.addEventListener("timeupdate", reflectAudioTimeChanges);
    const endTime = getEndTime();
    if (endTime) endTime.innerHTML = formatSeconds(audioFile.duration);
  });
}

export const safelyCloseAudio = () => {
  const closer = document.querySelector(CLOSE_PLAYER_BUTTON_SELECTOR);
  if (!closer) return;
  closer.dispatchEvent(new Event("click"));
}
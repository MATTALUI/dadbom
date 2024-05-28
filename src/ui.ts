import { getAudioFile } from "./state";

const AUDIO_PLAYER_BUTTON_SELECTOR = "button[aria-label='Audio Player']";
const CLOSE_PLAYER_BUTTON_SELECTOR = "button[aria-label='Close']";
const PLAY_BUTTON_SELECTOR = "button[aria-label='Play']";

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
    audioFile.play();
  });
}

export const getPlayButton = (): HTMLDivElement | null => {
  return document.querySelector(PLAY_BUTTON_SELECTOR);
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
    });
    // Add the intercept for the playbutton
    removePlayButtonListeners()
    addPlayButtonIntercepts();
  });
}
import { getBOMFileURL } from "./bofm";
import { getAudioFile, setAudioFile } from "./state";
import {
  disableAudioPlayerButton,
  addAudioToggleIntercepts,
  getAudioPlayerButton,
  enableAudioPlayerButton,
  getCloseButton,
  addAudioPlayerIntercepts,
} from "./ui"

(async () => {
  const initializeDadBOMPlayer = async () => {
    const audioPlayerButton = getAudioPlayerButton();
    const closeButton = getCloseButton();
    if (!audioPlayerButton && !closeButton) return;
    try {
      if (audioPlayerButton) disableAudioPlayerButton();
      const url = getBOMFileURL();
      if (!url) return;
      const audioFile = new Audio(url);
      await new Promise((resolve, reject) => {
        audioFile.addEventListener("canplay", () => {
          resolve(audioFile);
        });
        audioFile.addEventListener("error", () => {
          reject(`Unable to load Dad BOM audiofile: ${url}`);
        });
      });
      setAudioFile(audioFile);
      if (audioPlayerButton) addAudioToggleIntercepts();
      else addAudioPlayerIntercepts();
    } catch (e) {
      console.error(e);
    } finally {
      if (audioPlayerButton) enableAudioPlayerButton();
    }
  }

  // We have a mutation observer that watches the window's URL because the
  // scriptures part of the site is actually on a SPA, so when using some of the
  // sidebar links to change the scriptures don't actually trigger reloads in
  // the extension scripts
  let lastUrl = window.location.href;
  new MutationObserver(() => {
    const url = location.href;
    if (url !== lastUrl) {
      lastUrl = url;
      try {
        const audioFile = getAudioFile();
        if (!audioFile.paused) {
          audioFile.pause();
        }
      } catch (e) {
        console.error(e);
      }
      setTimeout(() => {
        initializeDadBOMPlayer();
      }, 100);
    }
  }).observe(document, { subtree: true, childList: true });
  // And the initial run
  initializeDadBOMPlayer();
})()
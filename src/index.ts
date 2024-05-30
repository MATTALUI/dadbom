import { setAudioFile } from "./state";
import { disableAudioPlayerButton, addAudioPlayerIntercepts, getAudioPlayerButton, enableAudioPlayerButton } from "./ui"

(async () => {
  const audioPlayerButton = getAudioPlayerButton();
  if (audioPlayerButton) {
    disableAudioPlayerButton();
    try {
      const url = "https://masterofnone-dev.s3.us-west-2.amazonaws.com/1ne1.mp3"
      const audioFile = new Audio(url);
      await new Promise((resolve, reject) => {
        audioFile.addEventListener("canplay", () => {
          resolve(audioFile);
        });
        // Add listener for audio failing to load
      });
      setAudioFile(audioFile);
      addAudioPlayerIntercepts();
    } catch (e) {
      console.error(e);
    } finally {
      enableAudioPlayerButton();
    }
  }
})()
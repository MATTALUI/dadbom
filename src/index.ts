import { hasAudioPlayer, loadAudioFile } from "./ui"

(async () => {
  console.log("Eveything is running!")

  if (hasAudioPlayer()) {
    try {
      const audioFile = await loadAudioFile();
      console.log("click to hear it all")
      document.addEventListener("click", () => {
        console.log("it's about to go down");

        audioFile.play();
      });

    } catch (e) {
      console.error("couldn't load audio");
      console.error(e);
    }
  }
})()
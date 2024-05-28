// example url
// https://www.churchofjesuschrist.org/study/scriptures/bofm/1-ne/1?lang=eng
// possible pattern matcher
// \/study\/scriptures\/bofm

export const getAudioPlayerButton = (): HTMLDivElement | null => {
  return document.querySelector("button[aria-label='Audio Player']");
}

export const hasAudioPlayer = (): boolean => {
  return !!getAudioPlayerButton();
}

export const loadAudioFile = async (): Promise<HTMLAudioElement> => {
  const url = "https://masterofnone-dev.s3.us-west-2.amazonaws.com/008-1+Ne.+1.mp3"
  const audio = new Audio(url);
  return new Promise((resolve, reject) => {
    if (!audio) return reject("Missing required data");

    audio.addEventListener("canplay", () => {
      resolve(audio);
    });
  });
}
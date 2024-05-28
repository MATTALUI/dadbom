type State = {
  audioFile: HTMLAudioElement | null;
  audioLoaded: boolean;
};

const state: State = {
  audioFile: null,
  audioLoaded: true,
};

export const getAudioFile = (): HTMLAudioElement => {
  if (!state.audioFile)
    throw new Error("Audio file has not been loaded yet");

  return state.audioFile;
}

export const setAudioFile = (audioFile: HTMLAudioElement) => {
  state.audioFile = audioFile;
  state.audioLoaded = true;
}
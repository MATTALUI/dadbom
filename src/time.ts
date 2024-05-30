export const formatSeconds = (seconds: number): string => {
  const min = Math.floor(seconds / 60);
  const sec = Math.floor(seconds - (min * 60));

  return `${min.toString().padStart(2, '0')}:${sec.toString().padStart(2, '0')}`;
}
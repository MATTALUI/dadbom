
const S3_BASE = "https://masterofnone-dev.s3.us-west-2.amazonaws.com/BOM"

const bookmap: Record<string, string> = {
  "1-ne": "00-1nephi",
  "2-ne": "01-2nephi",
  "jacob": "02-jacob",
  "enos": "03-enos",
  "jarom": "04-jarom",
  "omni": "05-omni",
  "w-of-m": "06-wom",
  "mosiah": "07-mosiah",
  "alma": "08-alma",
  "hel": "09-helaman",
  "3-ne": "10-3nephi",
  "4-ne": "11-4nephi",
  "morm": "12-mormon",
  "ether": "13-ether",
  "moro": "14-moroni",
  "": "15-preface",
}

export const getBOMFileURL = (): string | null => {
  const path = window.location.pathname
  const regex = /bofm\/.*\/\d{1,}/;
  const match = regex.exec(path);
  if (!match || match.length !== 1) return null;
  const bookKey = match[0].split('/')[1];
  const chapter = match[0].split('/')[2];
  const targetBook = bookmap[bookKey];

  const url = `${S3_BASE}/${targetBook}/${chapter}.mp3`;
  console.table({
    path,
    bookKey,
    chapter,
    targetBook,
    url,
  });

  return url;
}
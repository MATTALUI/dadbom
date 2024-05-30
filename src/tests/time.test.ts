import { formatSeconds } from "../time";

describe("formatSeconds", () => {
  it("correctly formats seconds into a time stamp", () => {
    expect(formatSeconds(0)).toEqual("00:00");
    expect(formatSeconds(6.9)).toEqual("00:06");
    expect(formatSeconds(6)).toEqual("00:06");
    expect(formatSeconds(30)).toEqual("00:30");
    expect(formatSeconds(60)).toEqual("01:00");
    expect(formatSeconds(59.651564)).toEqual("00:59")
    expect(formatSeconds(61)).toEqual("01:01");
    expect(formatSeconds(119.02)).toEqual("01:59");
    expect(formatSeconds(120)).toEqual("02:00");
    expect(formatSeconds(120.1)).toEqual("02:00");
    expect(formatSeconds(60 * 65)).toEqual("65:00");
  });
})
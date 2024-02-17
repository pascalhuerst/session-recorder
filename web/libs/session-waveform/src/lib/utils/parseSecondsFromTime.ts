export const parseSecondsFromTime = (value: string) => {
  const [h, m, s, ms] =
    value
      .match(/(\d{2}):(\d{2}):(\d{2})\.(\d{3})/)
      ?.slice(1, 5)
      .map((val) => parseInt(val)) || [];
  return h * 3600 + m * 60 + s + ms / 1000;
};

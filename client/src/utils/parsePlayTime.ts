const padLeft = (value: string | number, symbol: string, length: number) => {
  const diff = length - String(value).length;
  if (diff < 0) {
    return String(value);
  }

  return `${new Array(diff).fill(symbol, 0, diff)}${value}`;
};

export const parsePlayTime = (seconds: number) => {
  const date = new Date(seconds * 1000);

  return {
    hours: padLeft(date.getUTCHours(), "0", 2),
    minutes: padLeft(date.getUTCMinutes(), "0", 2),
    seconds: padLeft(date.getUTCSeconds(), "0", 2),
    milliseconds: padLeft(date.getUTCMilliseconds(), "0", 3)
  };
};
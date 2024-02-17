const padLeft = (value: string | number, symbol: string, length: number) => {
  const diff = length - value.toString().length;
  if (diff < 0) {
    return String(value);
  }

  const prefix = [...new Array(diff)].map(() => symbol).join('');
  return `${prefix}${value}`;
};

export const parseTimeFromSeconds = (seconds: number) => {
  const date = new Date(seconds * 1000);

  return {
    hours: padLeft(date.getUTCHours(), '0', 2),
    minutes: padLeft(date.getUTCMinutes(), '0', 2),
    seconds: padLeft(date.getUTCSeconds(), '0', 2),
    milliseconds: padLeft(date.getUTCMilliseconds(), '0', 3),
  };
};

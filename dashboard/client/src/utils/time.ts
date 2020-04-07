import moment from 'moment';

export const TIMESTAMP_FORMAT = 'YYYY-MM-DD HH:mm:ss';

export const normalizeDrift = (rawDrift: number): number => {
  if (rawDrift < 0) {
    return 0;
  }
  return rawDrift;
};

export const timestampToTimeString = (timestamp: number): string => {
  return moment(timestamp, 'X').format(TIMESTAMP_FORMAT);
};

export const stringToMoment = (timestamp: string): moment.Moment => {
  return moment(timestamp, TIMESTAMP_FORMAT);
};

export const durationToHuman = (duration: number): string => {
  const d = moment.duration(duration * 1000);
  return d.humanize();
};

const MINUTE = 60;
const HOUR = 60 * 60;
const DAY = 60 * 60 * 24;

const pluralizeDays = (days: number): string => {
  return `${days.toFixed(0)} day${days > 1 ? 's' : ''}`;
};

export const durationToHumanBeta = (duration: number): string => {
  if (duration === 0) {
    return '0s';
  } else if (duration < MINUTE) {
    // under a minute
    return `${duration.toFixed(2)}s`;
  } else if (duration < HOUR) {
    // under an hour
    const min = Math.floor(duration / MINUTE);
    const sec = duration - min * MINUTE;
    return `${min.toFixed(0)}m ${sec.toFixed(0)}s`;
  } else if (duration < DAY) {
    // 54,000
    // under a day
    const hours = Math.floor(duration / HOUR);
    const min = Math.floor((duration - hours * HOUR) / MINUTE);
    return `${hours.toFixed(0)}h ${min.toFixed(0)}m`;
  } else {
    // greater then a day
    const day = Math.floor(duration / DAY);
    const hour = Math.floor((duration - day * DAY) / HOUR);
    return `${pluralizeDays(day)} ${hour.toFixed(0)}h`;
  }
};

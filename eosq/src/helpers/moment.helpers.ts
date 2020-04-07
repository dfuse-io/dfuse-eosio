import moment from "moment-timezone"

// FIXME: Should we move all this somewhere else?
const guessedTz = moment.tz.guess()
const shortTimezoneString = moment.tz(guessedTz).format("z")

// Hackish as we customize the locale format ourself, but seems good enough for now
moment.locales().forEach((locale: string) =>
  moment.updateLocale(locale, {
    // @ts-ignore The `longDateFormat` spec can received a single key, TypeDefs does not allow it
    longDateFormat: {
      lll: moment
        .localeData(locale)
        .longDateFormat("lll")
        .replace(/:mm /, ":mm:ss ")
    }
  })
)

export function formatDateFromString(date: string | number | Date, utc: boolean): string {
  if (utc) {
    return moment.utc(date, "YYYY-MM-DD hh:mm:ss Z").format("YYYY-MM-DDTHH:mm:ss [UTC]")
  }

  return `${moment
    .utc(date, "YYYY-MM-DD hh:mm:ss Z")
    .local()
    .format("lll")} ${shortTimezoneString}`
}

export function blockTimeEstimate(headBlockNumber: number, blockNum: number, utc: boolean) {
  moment.tz.guess()

  const secondsAgo = (headBlockNumber - blockNum) / 2
  const roughBlockTime = parseInt(moment().format("X"), 10) - secondsAgo

  const date = moment.unix(roughBlockTime)
  if (utc) {
    return moment.utc(date).format("LLL UTC")
  }
  return date.format("lll")
}

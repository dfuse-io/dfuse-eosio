import { compactInteger, formatNumber as humanizeFormatNumber, toFixed } from "humanize-plus"
import moment from "moment-timezone"
import numeral from "numeral"
import { take as takeFirst, takeLast } from "ramda"
import { formatSI } from "./format-si-prefix"
import "moment-duration-format"
import { sha256 } from "js-sha256"

export const NBSP = "\u00a0"
export const ELLIPSIS = "\u2026"
export const INFINITY = "\u221e"
export const LONGDASH = "\u2014"
export const BULLET = "\u2022"

/**
 *
 *
 * Compact a string field into a more shorter form. From the
 * input `000210185aca9762f3906832b3ac983510862a04f2b6ee602619958ddccb5823`,
 * the resulting output of this function will be
 * `000210185aca9762f3906832b3ac983510862a...dccb5823`
 *
 * @param input The input to shorten
 */
export function compactString(input: number | string, first: number = 4, last: number = 2) {
  const id = input.toString()

  return [takeFirst(first, id), ELLIPSIS, takeLast(last, id)]
}

export function formatTransactionID(value: string) {
  return compactString(value, 10, 0)
}

export function truncateString(input: number | string, first: number) {
  const id = input.toString()
  if (id.length <= first) return [id, null]

  return [takeFirst(first, id), ELLIPSIS]
}

export function compactCount(input: number) {
  return compactInteger(input, 2)
}

export function microSecondsToSeconds(microseconds: number): number {
  return microseconds * 0.000001
}

export function getAmount(value: string): number {
  return parseFloat(value.split(" ")[0] || "0")
}

export function formatMicroseconds(microseconds: number) {
  return `${formatSI(microSecondsToSeconds(microseconds))}s`
}

export function formatBytes(value: number, limit: number = 0) {
  if (limit === 0) {
    return numeral(value)
      .format("0.0 b")
      .replace(".0", "")
  }

  if (value > limit) {
    return `${formatNumber(value)} B`
  }

  return numeral(value)
    .format("0.0 b")
    .replace(".0", "")
}

export function formatAmount(input: number) {
  return `$${toFixed(input, 2)}`
}

export function formatEosAmount(input: number) {
  return `${compactCount(input)} EOS`
}

export function formatNumber(input: number) {
  return humanizeFormatNumber(input)
}

export function formatPercentage(input: number, precision: number = 2) {
  const value = round(input, 7) <= 1.0 ? input * 100.0 : input

  return `${toFixed(value, precision)} %`
}

function round(value: number, decimals: number) {
  const integral = parseFloat(`${value.toString()}e${decimals.toString()}`)

  return Number(`${Math.round(integral)}e-${decimals}`)
}

export function formatDateTime(input: moment.Moment | Date | string | number) {
  return moment(input)
    .utc()
    .format("DD MMM YYYY hh:mm")
}

export function formatVariation(input: number) {
  if (input === 0) {
    return "0.00"
  }

  const value = toFixed(input, 2)

  return `${value}`
}

export function extractValueWithUnits(text: string) {
  return text.split(" ")
}

export function secondsToTime(seconds: number) {
  if (seconds <= 60) {
    return `${seconds} seconds`
  }
  return moment.duration(seconds, "seconds").format("Y [years] M [months] D [days] [+] h:mm:ss")
}

export function explodeJson(json: {} | string): string {
  if (!json) {
    return ""
  }

  if (typeof json === "string") {
    return json
  }

  return (
    Object.keys(json)
      .map((key: string) => {
        if (isLiteral(json[key])) {
          return `${key}: ${json[key]}`
        }

        if (Array.isArray(json[key])) {
          const entries = json[key].map((entry: any) => {
            return explodeJson(entry)
          })

          return `${key}: [ ${entries.join(", ")} ]`
        }

        const explodedJson = explodeJson(json[key])
        if (explodedJson.replace(/ /g, "").length === 0 || !explodedJson) {
          return ""
        }

        return `${key}: { ${explodeJson(json[key])} }`
      })
      .join(" ") || ""
  )
}

export function hex2sha256(hexData: string) {
  const matches = hexData.match(/[\da-f]{2}/gi)
  if (matches != null) {
    return sha256(new Uint8Array(matches.map((match) => parseInt(match, 16))))
  }

  return ""
}

export function hex2binary(hexData: string) {
  const matches = hexData.match(/[\da-f]{2}/gi)
  if (matches != null) {
    return new Uint8Array(matches.map((match) => parseInt(match, 16)))
  }

  return ""
}

function isLiteral(field: any) {
  return (
    typeof field === "string" ||
    typeof field === "number" ||
    field === null ||
    typeof field === "boolean"
  )
}

const REGEX_JAPANESE = /[\u3000-\u303f]|[\u3040-\u309f]|[\u30a0-\u30ff]|[\uff00-\uff9f]|[\u4e00-\u9faf]|[\u3400-\u4dbf]/
const REGEX_CHINESE = /[\u4e00-\u9fff]|[\u3400-\u4dbf]|[\u{20000}-\u{2a6df}]|[\u{2a700}-\u{2b73f}]|[\u{2b740}-\u{2b81f}]|[\u{2b820}-\u{2ceaf}]|[\uf900-\ufaff]|[\u3300-\u33ff]|[\ufe30-\ufe4f]|[\uf900-\ufaff]|[\u{2f800}-\u{2fa1f}]/u
const REGEX_KOREAN = /[\u3131-\uD79D]/giu

export function hasAsianCharacters(str: string) {
  return REGEX_JAPANESE.test(str) || REGEX_CHINESE.test(str) || REGEX_KOREAN.test(str)
}

export function humanizeSnakeCase(value: string) {
  const capitalizedValue = value.charAt(0).toUpperCase() + value.slice(1)
  return capitalizedValue.replace(/(_)/g, " ")
}

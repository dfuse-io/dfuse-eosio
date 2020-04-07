import {
  ELLIPSIS,
  explodeJson,
  formatPercentage,
  formatVariation,
  hex2sha256,
  secondsToTime,
  truncateString
} from "../formatters"

describe("explodeJson", () => {
  it("parse json", () => {
    const action = explodeJson({ data: { memo: "test" } })
    expect(explodeJson(action)).toEqual("data: { memo: test }")
  })

  it("parse json with arrays", () => {
    const action = explodeJson({ data: [{ memo: "test" }] })
    expect(explodeJson(action)).toEqual("data: [ memo: test ]")
  })

  it("parse nested json with arrays", () => {
    const action = explodeJson({ data: [{ memo: { foo: 100 } }, { memo: { foo: true } }] })
    expect(explodeJson(action)).toEqual("data: [ memo: { foo: 100 }, memo: { foo: true } ]")
  })
})

describe("truncateString", () => {
  it("should return the correct string output", () => {
    expect(truncateString("1.34567", 5)).toEqual(["1.345", ELLIPSIS])
    expect(truncateString("1.34", 2)).toEqual(["1.", ELLIPSIS])
    expect(truncateString("", 2)).toEqual(["", null])
  })
})

describe("formatPercentage", () => {
  it("should format a fraction to a percentage", () => {
    expect(formatPercentage(0.9)).toEqual("90.00 %")
    expect(formatPercentage(1)).toEqual("100.00 %")
    expect(formatPercentage(2)).toEqual("2.00 %")
  })
})

describe("secondsToTime", () => {
  it("should return the time in a days - hh:mm:ss format", () => {
    expect(secondsToTime(3000)).toEqual("50:00")
    expect(secondsToTime(3600 * 24)).toEqual("1 day + 0:00:00")
  })
})

describe("formatVariation", () => {
  it("should return the right variation string", () => {
    expect(formatVariation(0)).toEqual("0.00")
    expect(formatVariation(0.4444)).toEqual("0.44")
  })
})

describe("hex2sha256", () => {
  it("should return the right variation string", () => {
    expect(
      hex2sha256(
        "0061736d010000000125076000017e60027e7e0060027f7f017f6000017f60017f0060037e7e7e0060037f7f7f017f02280203656e760c63757272656e745f74696d65000003656e760d726571756972655f61757468320001030807020202030405060404017000000503010001079f0108066d656d6f72790200165f5a6571524b3131636865636b73756d32353653315f0002165f5a6571524b3131636865636b73756d31363053315f0003165f5a6e65524b3131636865636b73756d31363053315f0004036e6f770005305f5a4e35656f73696f3132726571756972655f6175746845524b4e535f31367065726d697373696f6e5f6c6576656c450006056170706c790007066d656d636d7000080a8e01070b002000200141201008450b0b002000200141201008450b0d0020002001412010084100470b0a00100042c0843d80a70b0e002000290300200029030810010b02000b4901037f4100210502402002450d000240034020002d0000220320012d00002204470d01200141016a2101200041016a21002002417f6a22020d000c020b0b200320046b21050b20050b0b0a010041040b0410400000"
      )
    ).toEqual("cb62eddcfde0ddbde5c00a87d102f6c4eaf5f13a112bda4d001868fbb29f7001")
  })
})

import { task } from "mobx-task"
import { debugLog } from "./logger"
import { requestProducerSchedule } from "../clients/rest/account"

export const fetchProducerSchedule = task(
  async () => {
    const producerScheduleResponse = await requestProducerSchedule()

    if (producerScheduleResponse) {
      debugLog(
        "Found schedule version [%s] via search API.",
        producerScheduleResponse.active.version,
        producerScheduleResponse
      )
      return producerScheduleResponse.active.producers
    }
    debugLog("Schedule not found anywhere.")
    return []
  },
  { swallow: true }
)

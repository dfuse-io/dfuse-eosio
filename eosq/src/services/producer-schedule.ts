import { task } from "mobx-task"
import { log } from "./logger"
import { requestProducerSchedule } from "../clients/rest/account"

export const fetchProducerSchedule = task(
  async () => {
    const producerScheduleResponse = await requestProducerSchedule()

    if (producerScheduleResponse) {
      log.info(
        "Found schedule version [%s] via search API.",
        producerScheduleResponse.active.version,
        producerScheduleResponse
      )
      return producerScheduleResponse.active.producers
    }
    log.info("Schedule not found anywhere.")
    return []
  },
  { swallow: true }
)

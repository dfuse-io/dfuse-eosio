/* eslint-disable */
// package: dashboard
// file: dashboard.proto

import * as jspb from "google-protobuf";
import * as google_protobuf_timestamp_pb from "google-protobuf/google/protobuf/timestamp_pb";

export class AppsListRequest extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AppsListRequest.AsObject;
  static toObject(includeInstance: boolean, msg: AppsListRequest): AppsListRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: AppsListRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AppsListRequest;
  static deserializeBinaryFromReader(message: AppsListRequest, reader: jspb.BinaryReader): AppsListRequest;
}

export namespace AppsListRequest {
  export type AsObject = {
  }
}

export class AppsListResponse extends jspb.Message {
  clearAppsList(): void;
  getAppsList(): Array<AppInfo>;
  setAppsList(value: Array<AppInfo>): void;
  addApps(value?: AppInfo, index?: number): AppInfo;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AppsListResponse.AsObject;
  static toObject(includeInstance: boolean, msg: AppsListResponse): AppsListResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: AppsListResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AppsListResponse;
  static deserializeBinaryFromReader(message: AppsListResponse, reader: jspb.BinaryReader): AppsListResponse;
}

export namespace AppsListResponse {
  export type AsObject = {
    appsList: Array<AppInfo.AsObject>,
  }
}

export class AppsInfoRequest extends jspb.Message {
  getFilterAppId(): string;
  setFilterAppId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AppsInfoRequest.AsObject;
  static toObject(includeInstance: boolean, msg: AppsInfoRequest): AppsInfoRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: AppsInfoRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AppsInfoRequest;
  static deserializeBinaryFromReader(message: AppsInfoRequest, reader: jspb.BinaryReader): AppsInfoRequest;
}

export namespace AppsInfoRequest {
  export type AsObject = {
    filterAppId: string,
  }
}

export class AppsInfoResponse extends jspb.Message {
  clearAppsList(): void;
  getAppsList(): Array<AppInfo>;
  setAppsList(value: Array<AppInfo>): void;
  addApps(value?: AppInfo, index?: number): AppInfo;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AppsInfoResponse.AsObject;
  static toObject(includeInstance: boolean, msg: AppsInfoResponse): AppsInfoResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: AppsInfoResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AppsInfoResponse;
  static deserializeBinaryFromReader(message: AppsInfoResponse, reader: jspb.BinaryReader): AppsInfoResponse;
}

export namespace AppsInfoResponse {
  export type AsObject = {
    appsList: Array<AppInfo.AsObject>,
  }
}

export class AppInfo extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getTitle(): string;
  setTitle(value: string): void;

  getDescription(): string;
  setDescription(value: string): void;

  getStatus(): AppStatusMap[keyof AppStatusMap];
  setStatus(value: AppStatusMap[keyof AppStatusMap]): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AppInfo.AsObject;
  static toObject(includeInstance: boolean, msg: AppInfo): AppInfo.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: AppInfo, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AppInfo;
  static deserializeBinaryFromReader(message: AppInfo, reader: jspb.BinaryReader): AppInfo;
}

export namespace AppInfo {
  export type AsObject = {
    id: string,
    title: string,
    description: string,
    status: AppStatusMap[keyof AppStatusMap],
  }
}

export class AppsMetricsRequest extends jspb.Message {
  getFilterAppId(): string;
  setFilterAppId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AppsMetricsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: AppsMetricsRequest): AppsMetricsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: AppsMetricsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AppsMetricsRequest;
  static deserializeBinaryFromReader(message: AppsMetricsRequest, reader: jspb.BinaryReader): AppsMetricsRequest;
}

export namespace AppsMetricsRequest {
  export type AsObject = {
    filterAppId: string,
  }
}

export class AppMetricsResponse extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  clearMetricsList(): void;
  getMetricsList(): Array<Metric>;
  setMetricsList(value: Array<Metric>): void;
  addMetrics(value?: Metric, index?: number): Metric;

  getTitle(): string;
  setTitle(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AppMetricsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: AppMetricsResponse): AppMetricsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: AppMetricsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AppMetricsResponse;
  static deserializeBinaryFromReader(message: AppMetricsResponse, reader: jspb.BinaryReader): AppMetricsResponse;
}

export namespace AppMetricsResponse {
  export type AsObject = {
    id: string,
    metricsList: Array<Metric.AsObject>,
    title: string,
  }
}

export class Metric extends jspb.Message {
  hasTimestamp(): boolean;
  clearTimestamp(): void;
  getTimestamp(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setTimestamp(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getValue(): number;
  setValue(value: number): void;

  getType(): MetricTypeMap[keyof MetricTypeMap];
  setType(value: MetricTypeMap[keyof MetricTypeMap]): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Metric.AsObject;
  static toObject(includeInstance: boolean, msg: Metric): Metric.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Metric, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Metric;
  static deserializeBinaryFromReader(message: Metric, reader: jspb.BinaryReader): Metric;
}

export namespace Metric {
  export type AsObject = {
    timestamp?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    value: number,
    type: MetricTypeMap[keyof MetricTypeMap],
  }
}

export class StartAppRequest extends jspb.Message {
  getAppId(): string;
  setAppId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StartAppRequest.AsObject;
  static toObject(includeInstance: boolean, msg: StartAppRequest): StartAppRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: StartAppRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StartAppRequest;
  static deserializeBinaryFromReader(message: StartAppRequest, reader: jspb.BinaryReader): StartAppRequest;
}

export namespace StartAppRequest {
  export type AsObject = {
    appId: string,
  }
}

export class StartAppResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StartAppResponse.AsObject;
  static toObject(includeInstance: boolean, msg: StartAppResponse): StartAppResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: StartAppResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StartAppResponse;
  static deserializeBinaryFromReader(message: StartAppResponse, reader: jspb.BinaryReader): StartAppResponse;
}

export namespace StartAppResponse {
  export type AsObject = {
  }
}

export class StopAppRequest extends jspb.Message {
  getAppId(): string;
  setAppId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StopAppRequest.AsObject;
  static toObject(includeInstance: boolean, msg: StopAppRequest): StopAppRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: StopAppRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StopAppRequest;
  static deserializeBinaryFromReader(message: StopAppRequest, reader: jspb.BinaryReader): StopAppRequest;
}

export namespace StopAppRequest {
  export type AsObject = {
    appId: string,
  }
}

export class StopAppResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StopAppResponse.AsObject;
  static toObject(includeInstance: boolean, msg: StopAppResponse): StopAppResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: StopAppResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StopAppResponse;
  static deserializeBinaryFromReader(message: StopAppResponse, reader: jspb.BinaryReader): StopAppResponse;
}

export namespace StopAppResponse {
  export type AsObject = {
  }
}

export class DmeshRequest extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DmeshRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DmeshRequest): DmeshRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DmeshRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DmeshRequest;
  static deserializeBinaryFromReader(message: DmeshRequest, reader: jspb.BinaryReader): DmeshRequest;
}

export namespace DmeshRequest {
  export type AsObject = {
  }
}

export class DmeshResponse extends jspb.Message {
  clearClientsList(): void;
  getClientsList(): Array<DmeshClient>;
  setClientsList(value: Array<DmeshClient>): void;
  addClients(value?: DmeshClient, index?: number): DmeshClient;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DmeshResponse.AsObject;
  static toObject(includeInstance: boolean, msg: DmeshResponse): DmeshResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DmeshResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DmeshResponse;
  static deserializeBinaryFromReader(message: DmeshResponse, reader: jspb.BinaryReader): DmeshResponse;
}

export namespace DmeshResponse {
  export type AsObject = {
    clientsList: Array<DmeshClient.AsObject>,
  }
}

export class DmeshClient extends jspb.Message {
  getHost(): string;
  setHost(value: string): void;

  getReady(): boolean;
  setReady(value: boolean): void;

  hasBoot(): boolean;
  clearBoot(): void;
  getBoot(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setBoot(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getServesResolveforks(): boolean;
  setServesResolveforks(value: boolean): void;

  getServesReversible(): boolean;
  setServesReversible(value: boolean): void;

  getHasMovingHead(): boolean;
  setHasMovingHead(value: boolean): void;

  getHasMovingTail(): boolean;
  setHasMovingTail(value: boolean): void;

  getShardSize(): number;
  setShardSize(value: number): void;

  getTierLevel(): number;
  setTierLevel(value: number): void;

  getTailBlockNum(): number;
  setTailBlockNum(value: number): void;

  getTailBlockId(): string;
  setTailBlockId(value: string): void;

  getIrrBlockNum(): number;
  setIrrBlockNum(value: number): void;

  getIrrBlockId(): string;
  setIrrBlockId(value: string): void;

  getHeadBlockNum(): number;
  setHeadBlockNum(value: number): void;

  getHeadBlockId(): string;
  setHeadBlockId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DmeshClient.AsObject;
  static toObject(includeInstance: boolean, msg: DmeshClient): DmeshClient.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DmeshClient, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DmeshClient;
  static deserializeBinaryFromReader(message: DmeshClient, reader: jspb.BinaryReader): DmeshClient;
}

export namespace DmeshClient {
  export type AsObject = {
    host: string,
    ready: boolean,
    boot?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    servesResolveforks: boolean,
    servesReversible: boolean,
    hasMovingHead: boolean,
    hasMovingTail: boolean,
    shardSize: number,
    tierLevel: number,
    tailBlockNum: number,
    tailBlockId: string,
    irrBlockNum: number,
    irrBlockId: string,
    headBlockNum: number,
    headBlockId: string,
  }
}

export interface AppStatusMap {
  NOTFOUND: 0;
  CREATED: 1;
  RUNNING: 2;
  WARNING: 3;
  STOPPED: 4;
}

export const AppStatus: AppStatusMap;

export interface MetricTypeMap {
  HEAD_BLOCK_TIME_DRIFT: 0;
  HEAD_BLOCK_NUMBER: 1;
}

export const MetricType: MetricTypeMap;


/* eslint-disable */
// package: dashboard
// file: dashboard.proto

import * as dashboard_pb from "./dashboard_pb";
import {grpc} from "@improbable-eng/grpc-web";

type DashboardAppsList = {
  readonly methodName: string;
  readonly service: typeof Dashboard;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof dashboard_pb.AppsListRequest;
  readonly responseType: typeof dashboard_pb.AppsListResponse;
};

type DashboardAppsInfo = {
  readonly methodName: string;
  readonly service: typeof Dashboard;
  readonly requestStream: false;
  readonly responseStream: true;
  readonly requestType: typeof dashboard_pb.AppsInfoRequest;
  readonly responseType: typeof dashboard_pb.AppsInfoResponse;
};

type DashboardAppsMetrics = {
  readonly methodName: string;
  readonly service: typeof Dashboard;
  readonly requestStream: false;
  readonly responseStream: true;
  readonly requestType: typeof dashboard_pb.AppsMetricsRequest;
  readonly responseType: typeof dashboard_pb.AppMetricsResponse;
};

type DashboardDmesh = {
  readonly methodName: string;
  readonly service: typeof Dashboard;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof dashboard_pb.DmeshRequest;
  readonly responseType: typeof dashboard_pb.DmeshResponse;
};

type DashboardStartApp = {
  readonly methodName: string;
  readonly service: typeof Dashboard;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof dashboard_pb.StartAppRequest;
  readonly responseType: typeof dashboard_pb.StartAppResponse;
};

type DashboardStopApp = {
  readonly methodName: string;
  readonly service: typeof Dashboard;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof dashboard_pb.StopAppRequest;
  readonly responseType: typeof dashboard_pb.StopAppResponse;
};

export class Dashboard {
  static readonly serviceName: string;
  static readonly AppsList: DashboardAppsList;
  static readonly AppsInfo: DashboardAppsInfo;
  static readonly AppsMetrics: DashboardAppsMetrics;
  static readonly Dmesh: DashboardDmesh;
  static readonly StartApp: DashboardStartApp;
  static readonly StopApp: DashboardStopApp;
}

export type ServiceError = { message: string, code: number; metadata: grpc.Metadata }
export type Status = { details: string, code: number; metadata: grpc.Metadata }

interface UnaryResponse {
  cancel(): void;
}
interface ResponseStream<T> {
  cancel(): void;
  on(type: 'data', handler: (message: T) => void): ResponseStream<T>;
  on(type: 'end', handler: (status?: Status) => void): ResponseStream<T>;
  on(type: 'status', handler: (status: Status) => void): ResponseStream<T>;
}
interface RequestStream<T> {
  write(message: T): RequestStream<T>;
  end(): void;
  cancel(): void;
  on(type: 'end', handler: (status?: Status) => void): RequestStream<T>;
  on(type: 'status', handler: (status: Status) => void): RequestStream<T>;
}
interface BidirectionalStream<ReqT, ResT> {
  write(message: ReqT): BidirectionalStream<ReqT, ResT>;
  end(): void;
  cancel(): void;
  on(type: 'data', handler: (message: ResT) => void): BidirectionalStream<ReqT, ResT>;
  on(type: 'end', handler: (status?: Status) => void): BidirectionalStream<ReqT, ResT>;
  on(type: 'status', handler: (status: Status) => void): BidirectionalStream<ReqT, ResT>;
}

export class DashboardClient {
  readonly serviceHost: string;

  constructor(serviceHost: string, options?: grpc.RpcOptions);
  appsList(
    requestMessage: dashboard_pb.AppsListRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: dashboard_pb.AppsListResponse|null) => void
  ): UnaryResponse;
  appsList(
    requestMessage: dashboard_pb.AppsListRequest,
    callback: (error: ServiceError|null, responseMessage: dashboard_pb.AppsListResponse|null) => void
  ): UnaryResponse;
  appsInfo(requestMessage: dashboard_pb.AppsInfoRequest, metadata?: grpc.Metadata): ResponseStream<dashboard_pb.AppsInfoResponse>;
  appsMetrics(requestMessage: dashboard_pb.AppsMetricsRequest, metadata?: grpc.Metadata): ResponseStream<dashboard_pb.AppMetricsResponse>;
  dmesh(
    requestMessage: dashboard_pb.DmeshRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: dashboard_pb.DmeshResponse|null) => void
  ): UnaryResponse;
  dmesh(
    requestMessage: dashboard_pb.DmeshRequest,
    callback: (error: ServiceError|null, responseMessage: dashboard_pb.DmeshResponse|null) => void
  ): UnaryResponse;
  startApp(
    requestMessage: dashboard_pb.StartAppRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: dashboard_pb.StartAppResponse|null) => void
  ): UnaryResponse;
  startApp(
    requestMessage: dashboard_pb.StartAppRequest,
    callback: (error: ServiceError|null, responseMessage: dashboard_pb.StartAppResponse|null) => void
  ): UnaryResponse;
  stopApp(
    requestMessage: dashboard_pb.StopAppRequest,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: dashboard_pb.StopAppResponse|null) => void
  ): UnaryResponse;
  stopApp(
    requestMessage: dashboard_pb.StopAppRequest,
    callback: (error: ServiceError|null, responseMessage: dashboard_pb.StopAppResponse|null) => void
  ): UnaryResponse;
}


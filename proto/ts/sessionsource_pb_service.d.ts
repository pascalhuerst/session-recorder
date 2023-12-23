// package: sessionsource
// file: sessionsource.proto

import * as sessionsource_pb from "./sessionsource_pb";
import {grpc} from "@improbable-eng/grpc-web";

type SessionSourceStreamOpenSessions = {
  readonly methodName: string;
  readonly service: typeof SessionSource;
  readonly requestStream: false;
  readonly responseStream: true;
  readonly requestType: typeof sessionsource_pb.StreamOpenSessionsRequest;
  readonly responseType: typeof sessionsource_pb.OpenSessions;
};

export class SessionSource {
  static readonly serviceName: string;
  static readonly StreamOpenSessions: SessionSourceStreamOpenSessions;
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

export class SessionSourceClient {
  readonly serviceHost: string;

  constructor(serviceHost: string, options?: grpc.RpcOptions);
  streamOpenSessions(requestMessage: sessionsource_pb.StreamOpenSessionsRequest, metadata?: grpc.Metadata): ResponseStream<sessionsource_pb.OpenSessions>;
}


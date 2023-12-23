/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import _m0 from "protobufjs/minimal";
import { AudioInputStatus, audioInputStatusFromJSON, audioInputStatusToJSON } from "./chunksink";
import { Duration } from "./google/protobuf/duration";
import { Timestamp } from "./google/protobuf/timestamp";

export const protobufPackage = "sessionsource";

export enum SessionState {
  SESSION_STATE_UNKNOWN = 0,
  SESSION_STATE_CREATED = 1,
  SESSION_STATE_RECORDING = 2,
  SESSION_STATE_FINISHED = 3,
  SESSION_STATE_ERROR = 4,
  UNRECOGNIZED = -1,
}

export function sessionStateFromJSON(object: any): SessionState {
  switch (object) {
    case 0:
    case "SESSION_STATE_UNKNOWN":
      return SessionState.SESSION_STATE_UNKNOWN;
    case 1:
    case "SESSION_STATE_CREATED":
      return SessionState.SESSION_STATE_CREATED;
    case 2:
    case "SESSION_STATE_RECORDING":
      return SessionState.SESSION_STATE_RECORDING;
    case 3:
    case "SESSION_STATE_FINISHED":
      return SessionState.SESSION_STATE_FINISHED;
    case 4:
    case "SESSION_STATE_ERROR":
      return SessionState.SESSION_STATE_ERROR;
    case -1:
    case "UNRECOGNIZED":
    default:
      return SessionState.UNRECOGNIZED;
  }
}

export function sessionStateToJSON(object: SessionState): string {
  switch (object) {
    case SessionState.SESSION_STATE_UNKNOWN:
      return "SESSION_STATE_UNKNOWN";
    case SessionState.SESSION_STATE_CREATED:
      return "SESSION_STATE_CREATED";
    case SessionState.SESSION_STATE_RECORDING:
      return "SESSION_STATE_RECORDING";
    case SessionState.SESSION_STATE_FINISHED:
      return "SESSION_STATE_FINISHED";
    case SessionState.SESSION_STATE_ERROR:
      return "SESSION_STATE_ERROR";
    case SessionState.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface StreamRecordersRequest {
}

export interface RecorderInfo {
  ID: string;
  name: string;
  audioInputStatus: AudioInputStatus;
}

export interface StreamSeesionRequst {
  recorderID: string;
}

export interface SessionInfo {
  ID: string;
  timeCreated: Date | undefined;
  timeFinished: Date | undefined;
  lifetime: Duration | undefined;
  audioFileName: string;
  waveformDataFile: string;
  keepSession: boolean;
  state: SessionState;
}

function createBaseStreamRecordersRequest(): StreamRecordersRequest {
  return {};
}

export const StreamRecordersRequest = {
  encode(_: StreamRecordersRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): StreamRecordersRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseStreamRecordersRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(_: any): StreamRecordersRequest {
    return {};
  },

  toJSON(_: StreamRecordersRequest): unknown {
    const obj: any = {};
    return obj;
  },

  create(base?: DeepPartial<StreamRecordersRequest>): StreamRecordersRequest {
    return StreamRecordersRequest.fromPartial(base ?? {});
  },
  fromPartial(_: DeepPartial<StreamRecordersRequest>): StreamRecordersRequest {
    const message = createBaseStreamRecordersRequest();
    return message;
  },
};

function createBaseRecorderInfo(): RecorderInfo {
  return { ID: "", name: "", audioInputStatus: 0 };
}

export const RecorderInfo = {
  encode(message: RecorderInfo, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.ID !== "") {
      writer.uint32(10).string(message.ID);
    }
    if (message.name !== "") {
      writer.uint32(18).string(message.name);
    }
    if (message.audioInputStatus !== 0) {
      writer.uint32(24).int32(message.audioInputStatus);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): RecorderInfo {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRecorderInfo();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.ID = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.name = reader.string();
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.audioInputStatus = reader.int32() as any;
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): RecorderInfo {
    return {
      ID: isSet(object.ID) ? globalThis.String(object.ID) : "",
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      audioInputStatus: isSet(object.audioInputStatus) ? audioInputStatusFromJSON(object.audioInputStatus) : 0,
    };
  },

  toJSON(message: RecorderInfo): unknown {
    const obj: any = {};
    if (message.ID !== "") {
      obj.ID = message.ID;
    }
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.audioInputStatus !== 0) {
      obj.audioInputStatus = audioInputStatusToJSON(message.audioInputStatus);
    }
    return obj;
  },

  create(base?: DeepPartial<RecorderInfo>): RecorderInfo {
    return RecorderInfo.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<RecorderInfo>): RecorderInfo {
    const message = createBaseRecorderInfo();
    message.ID = object.ID ?? "";
    message.name = object.name ?? "";
    message.audioInputStatus = object.audioInputStatus ?? 0;
    return message;
  },
};

function createBaseStreamSeesionRequst(): StreamSeesionRequst {
  return { recorderID: "" };
}

export const StreamSeesionRequst = {
  encode(message: StreamSeesionRequst, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.recorderID !== "") {
      writer.uint32(10).string(message.recorderID);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): StreamSeesionRequst {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseStreamSeesionRequst();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.recorderID = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): StreamSeesionRequst {
    return { recorderID: isSet(object.recorderID) ? globalThis.String(object.recorderID) : "" };
  },

  toJSON(message: StreamSeesionRequst): unknown {
    const obj: any = {};
    if (message.recorderID !== "") {
      obj.recorderID = message.recorderID;
    }
    return obj;
  },

  create(base?: DeepPartial<StreamSeesionRequst>): StreamSeesionRequst {
    return StreamSeesionRequst.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<StreamSeesionRequst>): StreamSeesionRequst {
    const message = createBaseStreamSeesionRequst();
    message.recorderID = object.recorderID ?? "";
    return message;
  },
};

function createBaseSessionInfo(): SessionInfo {
  return {
    ID: "",
    timeCreated: undefined,
    timeFinished: undefined,
    lifetime: undefined,
    audioFileName: "",
    waveformDataFile: "",
    keepSession: false,
    state: 0,
  };
}

export const SessionInfo = {
  encode(message: SessionInfo, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.ID !== "") {
      writer.uint32(10).string(message.ID);
    }
    if (message.timeCreated !== undefined) {
      Timestamp.encode(toTimestamp(message.timeCreated), writer.uint32(18).fork()).ldelim();
    }
    if (message.timeFinished !== undefined) {
      Timestamp.encode(toTimestamp(message.timeFinished), writer.uint32(26).fork()).ldelim();
    }
    if (message.lifetime !== undefined) {
      Duration.encode(message.lifetime, writer.uint32(34).fork()).ldelim();
    }
    if (message.audioFileName !== "") {
      writer.uint32(42).string(message.audioFileName);
    }
    if (message.waveformDataFile !== "") {
      writer.uint32(50).string(message.waveformDataFile);
    }
    if (message.keepSession === true) {
      writer.uint32(56).bool(message.keepSession);
    }
    if (message.state !== 0) {
      writer.uint32(64).int32(message.state);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SessionInfo {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSessionInfo();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.ID = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.timeCreated = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.timeFinished = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.lifetime = Duration.decode(reader, reader.uint32());
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.audioFileName = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.waveformDataFile = reader.string();
          continue;
        case 7:
          if (tag !== 56) {
            break;
          }

          message.keepSession = reader.bool();
          continue;
        case 8:
          if (tag !== 64) {
            break;
          }

          message.state = reader.int32() as any;
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SessionInfo {
    return {
      ID: isSet(object.ID) ? globalThis.String(object.ID) : "",
      timeCreated: isSet(object.timeCreated) ? fromJsonTimestamp(object.timeCreated) : undefined,
      timeFinished: isSet(object.timeFinished) ? fromJsonTimestamp(object.timeFinished) : undefined,
      lifetime: isSet(object.lifetime) ? Duration.fromJSON(object.lifetime) : undefined,
      audioFileName: isSet(object.audioFileName) ? globalThis.String(object.audioFileName) : "",
      waveformDataFile: isSet(object.waveformDataFile) ? globalThis.String(object.waveformDataFile) : "",
      keepSession: isSet(object.keepSession) ? globalThis.Boolean(object.keepSession) : false,
      state: isSet(object.state) ? sessionStateFromJSON(object.state) : 0,
    };
  },

  toJSON(message: SessionInfo): unknown {
    const obj: any = {};
    if (message.ID !== "") {
      obj.ID = message.ID;
    }
    if (message.timeCreated !== undefined) {
      obj.timeCreated = message.timeCreated.toISOString();
    }
    if (message.timeFinished !== undefined) {
      obj.timeFinished = message.timeFinished.toISOString();
    }
    if (message.lifetime !== undefined) {
      obj.lifetime = Duration.toJSON(message.lifetime);
    }
    if (message.audioFileName !== "") {
      obj.audioFileName = message.audioFileName;
    }
    if (message.waveformDataFile !== "") {
      obj.waveformDataFile = message.waveformDataFile;
    }
    if (message.keepSession === true) {
      obj.keepSession = message.keepSession;
    }
    if (message.state !== 0) {
      obj.state = sessionStateToJSON(message.state);
    }
    return obj;
  },

  create(base?: DeepPartial<SessionInfo>): SessionInfo {
    return SessionInfo.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SessionInfo>): SessionInfo {
    const message = createBaseSessionInfo();
    message.ID = object.ID ?? "";
    message.timeCreated = object.timeCreated ?? undefined;
    message.timeFinished = object.timeFinished ?? undefined;
    message.lifetime = (object.lifetime !== undefined && object.lifetime !== null)
      ? Duration.fromPartial(object.lifetime)
      : undefined;
    message.audioFileName = object.audioFileName ?? "";
    message.waveformDataFile = object.waveformDataFile ?? "";
    message.keepSession = object.keepSession ?? false;
    message.state = object.state ?? 0;
    return message;
  },
};

export type SessionSourceDefinition = typeof SessionSourceDefinition;
export const SessionSourceDefinition = {
  name: "SessionSource",
  fullName: "sessionsource.SessionSource",
  methods: {
    streamRecorders: {
      name: "StreamRecorders",
      requestType: StreamRecordersRequest,
      requestStream: false,
      responseType: RecorderInfo,
      responseStream: true,
      options: {},
    },
    streamSessions: {
      name: "StreamSessions",
      requestType: StreamSeesionRequst,
      requestStream: false,
      responseType: SessionInfo,
      responseStream: true,
      options: {},
    },
  },
} as const;

export interface SessionSourceServiceImplementation<CallContextExt = {}> {
  streamRecorders(
    request: StreamRecordersRequest,
    context: CallContext & CallContextExt,
  ): ServerStreamingMethodResult<DeepPartial<RecorderInfo>>;
  streamSessions(
    request: StreamSeesionRequst,
    context: CallContext & CallContextExt,
  ): ServerStreamingMethodResult<DeepPartial<SessionInfo>>;
}

export interface SessionSourceClient<CallOptionsExt = {}> {
  streamRecorders(
    request: DeepPartial<StreamRecordersRequest>,
    options?: CallOptions & CallOptionsExt,
  ): AsyncIterable<RecorderInfo>;
  streamSessions(
    request: DeepPartial<StreamSeesionRequst>,
    options?: CallOptions & CallOptionsExt,
  ): AsyncIterable<SessionInfo>;
}

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends globalThis.Array<infer U> ? globalThis.Array<DeepPartial<U>>
  : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function toTimestamp(date: Date): Timestamp {
  const seconds = date.getTime() / 1_000;
  const nanos = (date.getTime() % 1_000) * 1_000_000;
  return { seconds, nanos };
}

function fromTimestamp(t: Timestamp): Date {
  let millis = (t.seconds || 0) * 1_000;
  millis += (t.nanos || 0) / 1_000_000;
  return new globalThis.Date(millis);
}

function fromJsonTimestamp(o: any): Date {
  if (o instanceof globalThis.Date) {
    return o;
  } else if (typeof o === "string") {
    return new globalThis.Date(o);
  } else {
    return fromTimestamp(Timestamp.fromJSON(o));
  }
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}

export type ServerStreamingMethodResult<Response> = { [Symbol.asyncIterator](): AsyncIterator<Response, void> };

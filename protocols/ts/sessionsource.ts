/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import _m0 from "protobufjs/minimal";
import { RecorderStatus, Respone } from "./common";
import { Duration } from "./google/protobuf/duration";
import { Timestamp } from "./google/protobuf/timestamp";

export const protobufPackage = "sessionsource";

export enum SessionState {
  SESSION_STATE_UNKNOWN = 0,
  SESSION_STATE_RECORDING = 1,
  SESSION_STATE_FINISHED = 2,
  UNRECOGNIZED = -1,
}

export function sessionStateFromJSON(object: any): SessionState {
  switch (object) {
    case 0:
    case "SESSION_STATE_UNKNOWN":
      return SessionState.SESSION_STATE_UNKNOWN;
    case 1:
    case "SESSION_STATE_RECORDING":
      return SessionState.SESSION_STATE_RECORDING;
    case 2:
    case "SESSION_STATE_FINISHED":
      return SessionState.SESSION_STATE_FINISHED;
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
    case SessionState.SESSION_STATE_RECORDING:
      return "SESSION_STATE_RECORDING";
    case SessionState.SESSION_STATE_FINISHED:
      return "SESSION_STATE_FINISHED";
    case SessionState.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface StreamRecordersRequest {
}

export interface RecordereRemoved {
}

export interface Recorder {
  recorderID: string;
  recorderName: string;
  status?: RecorderStatus | undefined;
  removed?: RecordereRemoved | undefined;
}

export interface StreamSessionRequest {
  recorderID: string;
}

export interface SessionInfo {
  timeCreated: Date | undefined;
  timeFinished: Date | undefined;
  lifetime: Duration | undefined;
  name: string;
  audioFileName: string;
  waveformDataFile: string;
  keep: boolean;
  state: SessionState;
}

export interface SessionRemoved {
}

export interface Session {
  ID: string;
  updated?: SessionInfo | undefined;
  removed?: SessionRemoved | undefined;
}

export interface SetKeepSessionRequest {
  sessionID: string;
  keep: boolean;
}

export interface DeleteSessionRequest {
  sessionID: string;
}

export interface SetNameRequest {
  sessionID: string;
  name: string;
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

function createBaseRecordereRemoved(): RecordereRemoved {
  return {};
}

export const RecordereRemoved = {
  encode(_: RecordereRemoved, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): RecordereRemoved {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRecordereRemoved();
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

  fromJSON(_: any): RecordereRemoved {
    return {};
  },

  toJSON(_: RecordereRemoved): unknown {
    const obj: any = {};
    return obj;
  },

  create(base?: DeepPartial<RecordereRemoved>): RecordereRemoved {
    return RecordereRemoved.fromPartial(base ?? {});
  },
  fromPartial(_: DeepPartial<RecordereRemoved>): RecordereRemoved {
    const message = createBaseRecordereRemoved();
    return message;
  },
};

function createBaseRecorder(): Recorder {
  return { recorderID: "", recorderName: "", status: undefined, removed: undefined };
}

export const Recorder = {
  encode(message: Recorder, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.recorderID !== "") {
      writer.uint32(10).string(message.recorderID);
    }
    if (message.recorderName !== "") {
      writer.uint32(18).string(message.recorderName);
    }
    if (message.status !== undefined) {
      RecorderStatus.encode(message.status, writer.uint32(26).fork()).ldelim();
    }
    if (message.removed !== undefined) {
      RecordereRemoved.encode(message.removed, writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Recorder {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRecorder();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.recorderID = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.recorderName = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.status = RecorderStatus.decode(reader, reader.uint32());
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.removed = RecordereRemoved.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Recorder {
    return {
      recorderID: isSet(object.recorderID) ? globalThis.String(object.recorderID) : "",
      recorderName: isSet(object.recorderName) ? globalThis.String(object.recorderName) : "",
      status: isSet(object.status) ? RecorderStatus.fromJSON(object.status) : undefined,
      removed: isSet(object.removed) ? RecordereRemoved.fromJSON(object.removed) : undefined,
    };
  },

  toJSON(message: Recorder): unknown {
    const obj: any = {};
    if (message.recorderID !== "") {
      obj.recorderID = message.recorderID;
    }
    if (message.recorderName !== "") {
      obj.recorderName = message.recorderName;
    }
    if (message.status !== undefined) {
      obj.status = RecorderStatus.toJSON(message.status);
    }
    if (message.removed !== undefined) {
      obj.removed = RecordereRemoved.toJSON(message.removed);
    }
    return obj;
  },

  create(base?: DeepPartial<Recorder>): Recorder {
    return Recorder.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Recorder>): Recorder {
    const message = createBaseRecorder();
    message.recorderID = object.recorderID ?? "";
    message.recorderName = object.recorderName ?? "";
    message.status = (object.status !== undefined && object.status !== null)
      ? RecorderStatus.fromPartial(object.status)
      : undefined;
    message.removed = (object.removed !== undefined && object.removed !== null)
      ? RecordereRemoved.fromPartial(object.removed)
      : undefined;
    return message;
  },
};

function createBaseStreamSessionRequest(): StreamSessionRequest {
  return { recorderID: "" };
}

export const StreamSessionRequest = {
  encode(message: StreamSessionRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.recorderID !== "") {
      writer.uint32(10).string(message.recorderID);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): StreamSessionRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseStreamSessionRequest();
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

  fromJSON(object: any): StreamSessionRequest {
    return { recorderID: isSet(object.recorderID) ? globalThis.String(object.recorderID) : "" };
  },

  toJSON(message: StreamSessionRequest): unknown {
    const obj: any = {};
    if (message.recorderID !== "") {
      obj.recorderID = message.recorderID;
    }
    return obj;
  },

  create(base?: DeepPartial<StreamSessionRequest>): StreamSessionRequest {
    return StreamSessionRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<StreamSessionRequest>): StreamSessionRequest {
    const message = createBaseStreamSessionRequest();
    message.recorderID = object.recorderID ?? "";
    return message;
  },
};

function createBaseSessionInfo(): SessionInfo {
  return {
    timeCreated: undefined,
    timeFinished: undefined,
    lifetime: undefined,
    name: "",
    audioFileName: "",
    waveformDataFile: "",
    keep: false,
    state: 0,
  };
}

export const SessionInfo = {
  encode(message: SessionInfo, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.timeCreated !== undefined) {
      Timestamp.encode(toTimestamp(message.timeCreated), writer.uint32(18).fork()).ldelim();
    }
    if (message.timeFinished !== undefined) {
      Timestamp.encode(toTimestamp(message.timeFinished), writer.uint32(26).fork()).ldelim();
    }
    if (message.lifetime !== undefined) {
      Duration.encode(message.lifetime, writer.uint32(34).fork()).ldelim();
    }
    if (message.name !== "") {
      writer.uint32(42).string(message.name);
    }
    if (message.audioFileName !== "") {
      writer.uint32(50).string(message.audioFileName);
    }
    if (message.waveformDataFile !== "") {
      writer.uint32(58).string(message.waveformDataFile);
    }
    if (message.keep === true) {
      writer.uint32(64).bool(message.keep);
    }
    if (message.state !== 0) {
      writer.uint32(72).int32(message.state);
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

          message.name = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.audioFileName = reader.string();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.waveformDataFile = reader.string();
          continue;
        case 8:
          if (tag !== 64) {
            break;
          }

          message.keep = reader.bool();
          continue;
        case 9:
          if (tag !== 72) {
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
      timeCreated: isSet(object.timeCreated) ? fromJsonTimestamp(object.timeCreated) : undefined,
      timeFinished: isSet(object.timeFinished) ? fromJsonTimestamp(object.timeFinished) : undefined,
      lifetime: isSet(object.lifetime) ? Duration.fromJSON(object.lifetime) : undefined,
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      audioFileName: isSet(object.audioFileName) ? globalThis.String(object.audioFileName) : "",
      waveformDataFile: isSet(object.waveformDataFile) ? globalThis.String(object.waveformDataFile) : "",
      keep: isSet(object.keep) ? globalThis.Boolean(object.keep) : false,
      state: isSet(object.state) ? sessionStateFromJSON(object.state) : 0,
    };
  },

  toJSON(message: SessionInfo): unknown {
    const obj: any = {};
    if (message.timeCreated !== undefined) {
      obj.timeCreated = message.timeCreated.toISOString();
    }
    if (message.timeFinished !== undefined) {
      obj.timeFinished = message.timeFinished.toISOString();
    }
    if (message.lifetime !== undefined) {
      obj.lifetime = Duration.toJSON(message.lifetime);
    }
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.audioFileName !== "") {
      obj.audioFileName = message.audioFileName;
    }
    if (message.waveformDataFile !== "") {
      obj.waveformDataFile = message.waveformDataFile;
    }
    if (message.keep === true) {
      obj.keep = message.keep;
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
    message.timeCreated = object.timeCreated ?? undefined;
    message.timeFinished = object.timeFinished ?? undefined;
    message.lifetime = (object.lifetime !== undefined && object.lifetime !== null)
      ? Duration.fromPartial(object.lifetime)
      : undefined;
    message.name = object.name ?? "";
    message.audioFileName = object.audioFileName ?? "";
    message.waveformDataFile = object.waveformDataFile ?? "";
    message.keep = object.keep ?? false;
    message.state = object.state ?? 0;
    return message;
  },
};

function createBaseSessionRemoved(): SessionRemoved {
  return {};
}

export const SessionRemoved = {
  encode(_: SessionRemoved, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SessionRemoved {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSessionRemoved();
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

  fromJSON(_: any): SessionRemoved {
    return {};
  },

  toJSON(_: SessionRemoved): unknown {
    const obj: any = {};
    return obj;
  },

  create(base?: DeepPartial<SessionRemoved>): SessionRemoved {
    return SessionRemoved.fromPartial(base ?? {});
  },
  fromPartial(_: DeepPartial<SessionRemoved>): SessionRemoved {
    const message = createBaseSessionRemoved();
    return message;
  },
};

function createBaseSession(): Session {
  return { ID: "", updated: undefined, removed: undefined };
}

export const Session = {
  encode(message: Session, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.ID !== "") {
      writer.uint32(10).string(message.ID);
    }
    if (message.updated !== undefined) {
      SessionInfo.encode(message.updated, writer.uint32(18).fork()).ldelim();
    }
    if (message.removed !== undefined) {
      SessionRemoved.encode(message.removed, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Session {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSession();
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

          message.updated = SessionInfo.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.removed = SessionRemoved.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Session {
    return {
      ID: isSet(object.ID) ? globalThis.String(object.ID) : "",
      updated: isSet(object.updated) ? SessionInfo.fromJSON(object.updated) : undefined,
      removed: isSet(object.removed) ? SessionRemoved.fromJSON(object.removed) : undefined,
    };
  },

  toJSON(message: Session): unknown {
    const obj: any = {};
    if (message.ID !== "") {
      obj.ID = message.ID;
    }
    if (message.updated !== undefined) {
      obj.updated = SessionInfo.toJSON(message.updated);
    }
    if (message.removed !== undefined) {
      obj.removed = SessionRemoved.toJSON(message.removed);
    }
    return obj;
  },

  create(base?: DeepPartial<Session>): Session {
    return Session.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Session>): Session {
    const message = createBaseSession();
    message.ID = object.ID ?? "";
    message.updated = (object.updated !== undefined && object.updated !== null)
      ? SessionInfo.fromPartial(object.updated)
      : undefined;
    message.removed = (object.removed !== undefined && object.removed !== null)
      ? SessionRemoved.fromPartial(object.removed)
      : undefined;
    return message;
  },
};

function createBaseSetKeepSessionRequest(): SetKeepSessionRequest {
  return { sessionID: "", keep: false };
}

export const SetKeepSessionRequest = {
  encode(message: SetKeepSessionRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.sessionID !== "") {
      writer.uint32(10).string(message.sessionID);
    }
    if (message.keep === true) {
      writer.uint32(16).bool(message.keep);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SetKeepSessionRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSetKeepSessionRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.sessionID = reader.string();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.keep = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SetKeepSessionRequest {
    return {
      sessionID: isSet(object.sessionID) ? globalThis.String(object.sessionID) : "",
      keep: isSet(object.keep) ? globalThis.Boolean(object.keep) : false,
    };
  },

  toJSON(message: SetKeepSessionRequest): unknown {
    const obj: any = {};
    if (message.sessionID !== "") {
      obj.sessionID = message.sessionID;
    }
    if (message.keep === true) {
      obj.keep = message.keep;
    }
    return obj;
  },

  create(base?: DeepPartial<SetKeepSessionRequest>): SetKeepSessionRequest {
    return SetKeepSessionRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SetKeepSessionRequest>): SetKeepSessionRequest {
    const message = createBaseSetKeepSessionRequest();
    message.sessionID = object.sessionID ?? "";
    message.keep = object.keep ?? false;
    return message;
  },
};

function createBaseDeleteSessionRequest(): DeleteSessionRequest {
  return { sessionID: "" };
}

export const DeleteSessionRequest = {
  encode(message: DeleteSessionRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.sessionID !== "") {
      writer.uint32(10).string(message.sessionID);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DeleteSessionRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeleteSessionRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.sessionID = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DeleteSessionRequest {
    return { sessionID: isSet(object.sessionID) ? globalThis.String(object.sessionID) : "" };
  },

  toJSON(message: DeleteSessionRequest): unknown {
    const obj: any = {};
    if (message.sessionID !== "") {
      obj.sessionID = message.sessionID;
    }
    return obj;
  },

  create(base?: DeepPartial<DeleteSessionRequest>): DeleteSessionRequest {
    return DeleteSessionRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DeleteSessionRequest>): DeleteSessionRequest {
    const message = createBaseDeleteSessionRequest();
    message.sessionID = object.sessionID ?? "";
    return message;
  },
};

function createBaseSetNameRequest(): SetNameRequest {
  return { sessionID: "", name: "" };
}

export const SetNameRequest = {
  encode(message: SetNameRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.sessionID !== "") {
      writer.uint32(10).string(message.sessionID);
    }
    if (message.name !== "") {
      writer.uint32(18).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SetNameRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSetNameRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.sessionID = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.name = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SetNameRequest {
    return {
      sessionID: isSet(object.sessionID) ? globalThis.String(object.sessionID) : "",
      name: isSet(object.name) ? globalThis.String(object.name) : "",
    };
  },

  toJSON(message: SetNameRequest): unknown {
    const obj: any = {};
    if (message.sessionID !== "") {
      obj.sessionID = message.sessionID;
    }
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create(base?: DeepPartial<SetNameRequest>): SetNameRequest {
    return SetNameRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SetNameRequest>): SetNameRequest {
    const message = createBaseSetNameRequest();
    message.sessionID = object.sessionID ?? "";
    message.name = object.name ?? "";
    return message;
  },
};

export type SessionSourceDefinition = typeof SessionSourceDefinition;
export const SessionSourceDefinition = {
  name: "SessionSource",
  fullName: "sessionsource.SessionSource",
  methods: {
    /** Recorder RPC */
    streamRecorders: {
      name: "StreamRecorders",
      requestType: StreamRecordersRequest,
      requestStream: false,
      responseType: Recorder,
      responseStream: true,
      options: {},
    },
    /** Session RPC */
    streamSessions: {
      name: "StreamSessions",
      requestType: StreamSessionRequest,
      requestStream: false,
      responseType: Session,
      responseStream: true,
      options: {},
    },
    setKeepSession: {
      name: "SetKeepSession",
      requestType: SetKeepSessionRequest,
      requestStream: false,
      responseType: Respone,
      responseStream: false,
      options: {},
    },
    deleteSession: {
      name: "DeleteSession",
      requestType: DeleteSessionRequest,
      requestStream: false,
      responseType: Respone,
      responseStream: false,
      options: {},
    },
    setName: {
      name: "SetName",
      requestType: SetNameRequest,
      requestStream: false,
      responseType: Respone,
      responseStream: false,
      options: {},
    },
  },
} as const;

export interface SessionSourceServiceImplementation<CallContextExt = {}> {
  /** Recorder RPC */
  streamRecorders(
    request: StreamRecordersRequest,
    context: CallContext & CallContextExt,
  ): ServerStreamingMethodResult<DeepPartial<Recorder>>;
  /** Session RPC */
  streamSessions(
    request: StreamSessionRequest,
    context: CallContext & CallContextExt,
  ): ServerStreamingMethodResult<DeepPartial<Session>>;
  setKeepSession(request: SetKeepSessionRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Respone>>;
  deleteSession(request: DeleteSessionRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Respone>>;
  setName(request: SetNameRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Respone>>;
}

export interface SessionSourceClient<CallOptionsExt = {}> {
  /** Recorder RPC */
  streamRecorders(
    request: DeepPartial<StreamRecordersRequest>,
    options?: CallOptions & CallOptionsExt,
  ): AsyncIterable<Recorder>;
  /** Session RPC */
  streamSessions(
    request: DeepPartial<StreamSessionRequest>,
    options?: CallOptions & CallOptionsExt,
  ): AsyncIterable<Session>;
  setKeepSession(request: DeepPartial<SetKeepSessionRequest>, options?: CallOptions & CallOptionsExt): Promise<Respone>;
  deleteSession(request: DeepPartial<DeleteSessionRequest>, options?: CallOptions & CallOptionsExt): Promise<Respone>;
  setName(request: DeepPartial<SetNameRequest>, options?: CallOptions & CallOptionsExt): Promise<Respone>;
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

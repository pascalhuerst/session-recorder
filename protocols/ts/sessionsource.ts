/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import _m0 from "protobufjs/minimal";
import { RecorderStatus, Respone } from "./common";
import { Duration } from "./google/protobuf/duration";
import { Timestamp } from "./google/protobuf/timestamp";

export const protobufPackage = "sessionsource";

export enum SegmentState {
  SEGMENT_STATE_UNKNOWN = 0,
  SEGMENT_STATE_RENDERING = 1,
  SEGMENT_STATE_FINISHED = 2,
  SEGMENT_STATE_ERROR = 3,
  UNRECOGNIZED = -1,
}

export function segmentStateFromJSON(object: any): SegmentState {
  switch (object) {
    case 0:
    case "SEGMENT_STATE_UNKNOWN":
      return SegmentState.SEGMENT_STATE_UNKNOWN;
    case 1:
    case "SEGMENT_STATE_RENDERING":
      return SegmentState.SEGMENT_STATE_RENDERING;
    case 2:
    case "SEGMENT_STATE_FINISHED":
      return SegmentState.SEGMENT_STATE_FINISHED;
    case 3:
    case "SEGMENT_STATE_ERROR":
      return SegmentState.SEGMENT_STATE_ERROR;
    case -1:
    case "UNRECOGNIZED":
    default:
      return SegmentState.UNRECOGNIZED;
  }
}

export function segmentStateToJSON(object: SegmentState): string {
  switch (object) {
    case SegmentState.SEGMENT_STATE_UNKNOWN:
      return "SEGMENT_STATE_UNKNOWN";
    case SegmentState.SEGMENT_STATE_RENDERING:
      return "SEGMENT_STATE_RENDERING";
    case SegmentState.SEGMENT_STATE_FINISHED:
      return "SEGMENT_STATE_FINISHED";
    case SegmentState.SEGMENT_STATE_ERROR:
      return "SEGMENT_STATE_ERROR";
    case SegmentState.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

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

export interface SegmentInfo {
  timeStart: Date | undefined;
  timeEnd: Date | undefined;
  name: string;
  state: SegmentState;
}

export interface SegmentRemoved {
}

export interface Segment {
  segmentID: string;
  updated?: SegmentInfo | undefined;
  removed?: SegmentRemoved | undefined;
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
  segments: Segment[];
}

export interface SessionRemoved {
}

export interface Session {
  ID: string;
  updated?: SessionInfo | undefined;
  removed?: SessionRemoved | undefined;
}

export interface SetKeepSessionRequest {
  recorderID: string;
  sessionID: string;
  keep: boolean;
}

export interface DeleteSessionRequest {
  recorderID: string;
  sessionID: string;
}

export interface SetNameRequest {
  recorderID: string;
  sessionID: string;
  name: string;
}

export interface CreateSegmentRequest {
  recorderID: string;
  sessionID: string;
  segmentID: string;
  info: SegmentInfo | undefined;
}

export interface UpdateSegmentRequest {
  recorderID: string;
  sessionID: string;
  segmentID: string;
  info: SegmentInfo | undefined;
}

export interface DeleteSegmentRequest {
  recorderID: string;
  sessionID: string;
  segmentID: string;
}

export interface RenderSegmentRequest {
  recorderID: string;
  sessionID: string;
  segmentID: string;
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

function createBaseSegmentInfo(): SegmentInfo {
  return { timeStart: undefined, timeEnd: undefined, name: "", state: 0 };
}

export const SegmentInfo = {
  encode(message: SegmentInfo, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.timeStart !== undefined) {
      Timestamp.encode(toTimestamp(message.timeStart), writer.uint32(10).fork()).ldelim();
    }
    if (message.timeEnd !== undefined) {
      Timestamp.encode(toTimestamp(message.timeEnd), writer.uint32(18).fork()).ldelim();
    }
    if (message.name !== "") {
      writer.uint32(26).string(message.name);
    }
    if (message.state !== 0) {
      writer.uint32(32).int32(message.state);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SegmentInfo {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSegmentInfo();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.timeStart = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.timeEnd = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.name = reader.string();
          continue;
        case 4:
          if (tag !== 32) {
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

  fromJSON(object: any): SegmentInfo {
    return {
      timeStart: isSet(object.timeStart) ? fromJsonTimestamp(object.timeStart) : undefined,
      timeEnd: isSet(object.timeEnd) ? fromJsonTimestamp(object.timeEnd) : undefined,
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      state: isSet(object.state) ? segmentStateFromJSON(object.state) : 0,
    };
  },

  toJSON(message: SegmentInfo): unknown {
    const obj: any = {};
    if (message.timeStart !== undefined) {
      obj.timeStart = message.timeStart.toISOString();
    }
    if (message.timeEnd !== undefined) {
      obj.timeEnd = message.timeEnd.toISOString();
    }
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.state !== 0) {
      obj.state = segmentStateToJSON(message.state);
    }
    return obj;
  },

  create(base?: DeepPartial<SegmentInfo>): SegmentInfo {
    return SegmentInfo.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SegmentInfo>): SegmentInfo {
    const message = createBaseSegmentInfo();
    message.timeStart = object.timeStart ?? undefined;
    message.timeEnd = object.timeEnd ?? undefined;
    message.name = object.name ?? "";
    message.state = object.state ?? 0;
    return message;
  },
};

function createBaseSegmentRemoved(): SegmentRemoved {
  return {};
}

export const SegmentRemoved = {
  encode(_: SegmentRemoved, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SegmentRemoved {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSegmentRemoved();
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

  fromJSON(_: any): SegmentRemoved {
    return {};
  },

  toJSON(_: SegmentRemoved): unknown {
    const obj: any = {};
    return obj;
  },

  create(base?: DeepPartial<SegmentRemoved>): SegmentRemoved {
    return SegmentRemoved.fromPartial(base ?? {});
  },
  fromPartial(_: DeepPartial<SegmentRemoved>): SegmentRemoved {
    const message = createBaseSegmentRemoved();
    return message;
  },
};

function createBaseSegment(): Segment {
  return { segmentID: "", updated: undefined, removed: undefined };
}

export const Segment = {
  encode(message: Segment, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.segmentID !== "") {
      writer.uint32(10).string(message.segmentID);
    }
    if (message.updated !== undefined) {
      SegmentInfo.encode(message.updated, writer.uint32(18).fork()).ldelim();
    }
    if (message.removed !== undefined) {
      SegmentRemoved.encode(message.removed, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Segment {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSegment();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.segmentID = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.updated = SegmentInfo.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.removed = SegmentRemoved.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Segment {
    return {
      segmentID: isSet(object.segmentID) ? globalThis.String(object.segmentID) : "",
      updated: isSet(object.updated) ? SegmentInfo.fromJSON(object.updated) : undefined,
      removed: isSet(object.removed) ? SegmentRemoved.fromJSON(object.removed) : undefined,
    };
  },

  toJSON(message: Segment): unknown {
    const obj: any = {};
    if (message.segmentID !== "") {
      obj.segmentID = message.segmentID;
    }
    if (message.updated !== undefined) {
      obj.updated = SegmentInfo.toJSON(message.updated);
    }
    if (message.removed !== undefined) {
      obj.removed = SegmentRemoved.toJSON(message.removed);
    }
    return obj;
  },

  create(base?: DeepPartial<Segment>): Segment {
    return Segment.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Segment>): Segment {
    const message = createBaseSegment();
    message.segmentID = object.segmentID ?? "";
    message.updated = (object.updated !== undefined && object.updated !== null)
      ? SegmentInfo.fromPartial(object.updated)
      : undefined;
    message.removed = (object.removed !== undefined && object.removed !== null)
      ? SegmentRemoved.fromPartial(object.removed)
      : undefined;
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
    segments: [],
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
    for (const v of message.segments) {
      Segment.encode(v!, writer.uint32(82).fork()).ldelim();
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
        case 10:
          if (tag !== 82) {
            break;
          }

          message.segments.push(Segment.decode(reader, reader.uint32()));
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
      segments: globalThis.Array.isArray(object?.segments) ? object.segments.map((e: any) => Segment.fromJSON(e)) : [],
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
    if (message.segments?.length) {
      obj.segments = message.segments.map((e) => Segment.toJSON(e));
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
    message.segments = object.segments?.map((e) => Segment.fromPartial(e)) || [];
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
  return { recorderID: "", sessionID: "", keep: false };
}

export const SetKeepSessionRequest = {
  encode(message: SetKeepSessionRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.recorderID !== "") {
      writer.uint32(10).string(message.recorderID);
    }
    if (message.sessionID !== "") {
      writer.uint32(18).string(message.sessionID);
    }
    if (message.keep === true) {
      writer.uint32(24).bool(message.keep);
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

          message.recorderID = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.sessionID = reader.string();
          continue;
        case 3:
          if (tag !== 24) {
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
      recorderID: isSet(object.recorderID) ? globalThis.String(object.recorderID) : "",
      sessionID: isSet(object.sessionID) ? globalThis.String(object.sessionID) : "",
      keep: isSet(object.keep) ? globalThis.Boolean(object.keep) : false,
    };
  },

  toJSON(message: SetKeepSessionRequest): unknown {
    const obj: any = {};
    if (message.recorderID !== "") {
      obj.recorderID = message.recorderID;
    }
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
    message.recorderID = object.recorderID ?? "";
    message.sessionID = object.sessionID ?? "";
    message.keep = object.keep ?? false;
    return message;
  },
};

function createBaseDeleteSessionRequest(): DeleteSessionRequest {
  return { recorderID: "", sessionID: "" };
}

export const DeleteSessionRequest = {
  encode(message: DeleteSessionRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.recorderID !== "") {
      writer.uint32(10).string(message.recorderID);
    }
    if (message.sessionID !== "") {
      writer.uint32(18).string(message.sessionID);
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

          message.recorderID = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
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
    return {
      recorderID: isSet(object.recorderID) ? globalThis.String(object.recorderID) : "",
      sessionID: isSet(object.sessionID) ? globalThis.String(object.sessionID) : "",
    };
  },

  toJSON(message: DeleteSessionRequest): unknown {
    const obj: any = {};
    if (message.recorderID !== "") {
      obj.recorderID = message.recorderID;
    }
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
    message.recorderID = object.recorderID ?? "";
    message.sessionID = object.sessionID ?? "";
    return message;
  },
};

function createBaseSetNameRequest(): SetNameRequest {
  return { recorderID: "", sessionID: "", name: "" };
}

export const SetNameRequest = {
  encode(message: SetNameRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.recorderID !== "") {
      writer.uint32(10).string(message.recorderID);
    }
    if (message.sessionID !== "") {
      writer.uint32(18).string(message.sessionID);
    }
    if (message.name !== "") {
      writer.uint32(26).string(message.name);
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

          message.recorderID = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.sessionID = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
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
      recorderID: isSet(object.recorderID) ? globalThis.String(object.recorderID) : "",
      sessionID: isSet(object.sessionID) ? globalThis.String(object.sessionID) : "",
      name: isSet(object.name) ? globalThis.String(object.name) : "",
    };
  },

  toJSON(message: SetNameRequest): unknown {
    const obj: any = {};
    if (message.recorderID !== "") {
      obj.recorderID = message.recorderID;
    }
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
    message.recorderID = object.recorderID ?? "";
    message.sessionID = object.sessionID ?? "";
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseCreateSegmentRequest(): CreateSegmentRequest {
  return { recorderID: "", sessionID: "", segmentID: "", info: undefined };
}

export const CreateSegmentRequest = {
  encode(message: CreateSegmentRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.recorderID !== "") {
      writer.uint32(10).string(message.recorderID);
    }
    if (message.sessionID !== "") {
      writer.uint32(18).string(message.sessionID);
    }
    if (message.segmentID !== "") {
      writer.uint32(26).string(message.segmentID);
    }
    if (message.info !== undefined) {
      SegmentInfo.encode(message.info, writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CreateSegmentRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreateSegmentRequest();
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

          message.sessionID = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.segmentID = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.info = SegmentInfo.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): CreateSegmentRequest {
    return {
      recorderID: isSet(object.recorderID) ? globalThis.String(object.recorderID) : "",
      sessionID: isSet(object.sessionID) ? globalThis.String(object.sessionID) : "",
      segmentID: isSet(object.segmentID) ? globalThis.String(object.segmentID) : "",
      info: isSet(object.info) ? SegmentInfo.fromJSON(object.info) : undefined,
    };
  },

  toJSON(message: CreateSegmentRequest): unknown {
    const obj: any = {};
    if (message.recorderID !== "") {
      obj.recorderID = message.recorderID;
    }
    if (message.sessionID !== "") {
      obj.sessionID = message.sessionID;
    }
    if (message.segmentID !== "") {
      obj.segmentID = message.segmentID;
    }
    if (message.info !== undefined) {
      obj.info = SegmentInfo.toJSON(message.info);
    }
    return obj;
  },

  create(base?: DeepPartial<CreateSegmentRequest>): CreateSegmentRequest {
    return CreateSegmentRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<CreateSegmentRequest>): CreateSegmentRequest {
    const message = createBaseCreateSegmentRequest();
    message.recorderID = object.recorderID ?? "";
    message.sessionID = object.sessionID ?? "";
    message.segmentID = object.segmentID ?? "";
    message.info = (object.info !== undefined && object.info !== null)
      ? SegmentInfo.fromPartial(object.info)
      : undefined;
    return message;
  },
};

function createBaseUpdateSegmentRequest(): UpdateSegmentRequest {
  return { recorderID: "", sessionID: "", segmentID: "", info: undefined };
}

export const UpdateSegmentRequest = {
  encode(message: UpdateSegmentRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.recorderID !== "") {
      writer.uint32(10).string(message.recorderID);
    }
    if (message.sessionID !== "") {
      writer.uint32(18).string(message.sessionID);
    }
    if (message.segmentID !== "") {
      writer.uint32(26).string(message.segmentID);
    }
    if (message.info !== undefined) {
      SegmentInfo.encode(message.info, writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateSegmentRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateSegmentRequest();
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

          message.sessionID = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.segmentID = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.info = SegmentInfo.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): UpdateSegmentRequest {
    return {
      recorderID: isSet(object.recorderID) ? globalThis.String(object.recorderID) : "",
      sessionID: isSet(object.sessionID) ? globalThis.String(object.sessionID) : "",
      segmentID: isSet(object.segmentID) ? globalThis.String(object.segmentID) : "",
      info: isSet(object.info) ? SegmentInfo.fromJSON(object.info) : undefined,
    };
  },

  toJSON(message: UpdateSegmentRequest): unknown {
    const obj: any = {};
    if (message.recorderID !== "") {
      obj.recorderID = message.recorderID;
    }
    if (message.sessionID !== "") {
      obj.sessionID = message.sessionID;
    }
    if (message.segmentID !== "") {
      obj.segmentID = message.segmentID;
    }
    if (message.info !== undefined) {
      obj.info = SegmentInfo.toJSON(message.info);
    }
    return obj;
  },

  create(base?: DeepPartial<UpdateSegmentRequest>): UpdateSegmentRequest {
    return UpdateSegmentRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<UpdateSegmentRequest>): UpdateSegmentRequest {
    const message = createBaseUpdateSegmentRequest();
    message.recorderID = object.recorderID ?? "";
    message.sessionID = object.sessionID ?? "";
    message.segmentID = object.segmentID ?? "";
    message.info = (object.info !== undefined && object.info !== null)
      ? SegmentInfo.fromPartial(object.info)
      : undefined;
    return message;
  },
};

function createBaseDeleteSegmentRequest(): DeleteSegmentRequest {
  return { recorderID: "", sessionID: "", segmentID: "" };
}

export const DeleteSegmentRequest = {
  encode(message: DeleteSegmentRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.recorderID !== "") {
      writer.uint32(10).string(message.recorderID);
    }
    if (message.sessionID !== "") {
      writer.uint32(18).string(message.sessionID);
    }
    if (message.segmentID !== "") {
      writer.uint32(26).string(message.segmentID);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DeleteSegmentRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeleteSegmentRequest();
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

          message.sessionID = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.segmentID = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DeleteSegmentRequest {
    return {
      recorderID: isSet(object.recorderID) ? globalThis.String(object.recorderID) : "",
      sessionID: isSet(object.sessionID) ? globalThis.String(object.sessionID) : "",
      segmentID: isSet(object.segmentID) ? globalThis.String(object.segmentID) : "",
    };
  },

  toJSON(message: DeleteSegmentRequest): unknown {
    const obj: any = {};
    if (message.recorderID !== "") {
      obj.recorderID = message.recorderID;
    }
    if (message.sessionID !== "") {
      obj.sessionID = message.sessionID;
    }
    if (message.segmentID !== "") {
      obj.segmentID = message.segmentID;
    }
    return obj;
  },

  create(base?: DeepPartial<DeleteSegmentRequest>): DeleteSegmentRequest {
    return DeleteSegmentRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DeleteSegmentRequest>): DeleteSegmentRequest {
    const message = createBaseDeleteSegmentRequest();
    message.recorderID = object.recorderID ?? "";
    message.sessionID = object.sessionID ?? "";
    message.segmentID = object.segmentID ?? "";
    return message;
  },
};

function createBaseRenderSegmentRequest(): RenderSegmentRequest {
  return { recorderID: "", sessionID: "", segmentID: "" };
}

export const RenderSegmentRequest = {
  encode(message: RenderSegmentRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.recorderID !== "") {
      writer.uint32(10).string(message.recorderID);
    }
    if (message.sessionID !== "") {
      writer.uint32(18).string(message.sessionID);
    }
    if (message.segmentID !== "") {
      writer.uint32(26).string(message.segmentID);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): RenderSegmentRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRenderSegmentRequest();
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

          message.sessionID = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.segmentID = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): RenderSegmentRequest {
    return {
      recorderID: isSet(object.recorderID) ? globalThis.String(object.recorderID) : "",
      sessionID: isSet(object.sessionID) ? globalThis.String(object.sessionID) : "",
      segmentID: isSet(object.segmentID) ? globalThis.String(object.segmentID) : "",
    };
  },

  toJSON(message: RenderSegmentRequest): unknown {
    const obj: any = {};
    if (message.recorderID !== "") {
      obj.recorderID = message.recorderID;
    }
    if (message.sessionID !== "") {
      obj.sessionID = message.sessionID;
    }
    if (message.segmentID !== "") {
      obj.segmentID = message.segmentID;
    }
    return obj;
  },

  create(base?: DeepPartial<RenderSegmentRequest>): RenderSegmentRequest {
    return RenderSegmentRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<RenderSegmentRequest>): RenderSegmentRequest {
    const message = createBaseRenderSegmentRequest();
    message.recorderID = object.recorderID ?? "";
    message.sessionID = object.sessionID ?? "";
    message.segmentID = object.segmentID ?? "";
    return message;
  },
};

export type SessionSourceDefinition = typeof SessionSourceDefinition;
export const SessionSourceDefinition = {
  name: "SessionSource",
  fullName: "sessionsource.SessionSource",
  methods: {
    /** Stream */
    streamRecorders: {
      name: "StreamRecorders",
      requestType: StreamRecordersRequest,
      requestStream: false,
      responseType: Recorder,
      responseStream: true,
      options: {},
    },
    streamSessions: {
      name: "StreamSessions",
      requestType: StreamSessionRequest,
      requestStream: false,
      responseType: Session,
      responseStream: true,
      options: {},
    },
    /** Unary */
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
    createSegment: {
      name: "CreateSegment",
      requestType: CreateSegmentRequest,
      requestStream: false,
      responseType: Respone,
      responseStream: false,
      options: {},
    },
    deleteSegment: {
      name: "DeleteSegment",
      requestType: DeleteSegmentRequest,
      requestStream: false,
      responseType: Respone,
      responseStream: false,
      options: {},
    },
    renderSegment: {
      name: "RenderSegment",
      requestType: RenderSegmentRequest,
      requestStream: false,
      responseType: Respone,
      responseStream: false,
      options: {},
    },
    updateSegment: {
      name: "UpdateSegment",
      requestType: UpdateSegmentRequest,
      requestStream: false,
      responseType: Respone,
      responseStream: false,
      options: {},
    },
  },
} as const;

export interface SessionSourceServiceImplementation<CallContextExt = {}> {
  /** Stream */
  streamRecorders(
    request: StreamRecordersRequest,
    context: CallContext & CallContextExt,
  ): ServerStreamingMethodResult<DeepPartial<Recorder>>;
  streamSessions(
    request: StreamSessionRequest,
    context: CallContext & CallContextExt,
  ): ServerStreamingMethodResult<DeepPartial<Session>>;
  /** Unary */
  setKeepSession(request: SetKeepSessionRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Respone>>;
  deleteSession(request: DeleteSessionRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Respone>>;
  setName(request: SetNameRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Respone>>;
  createSegment(request: CreateSegmentRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Respone>>;
  deleteSegment(request: DeleteSegmentRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Respone>>;
  renderSegment(request: RenderSegmentRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Respone>>;
  updateSegment(request: UpdateSegmentRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Respone>>;
}

export interface SessionSourceClient<CallOptionsExt = {}> {
  /** Stream */
  streamRecorders(
    request: DeepPartial<StreamRecordersRequest>,
    options?: CallOptions & CallOptionsExt,
  ): AsyncIterable<Recorder>;
  streamSessions(
    request: DeepPartial<StreamSessionRequest>,
    options?: CallOptions & CallOptionsExt,
  ): AsyncIterable<Session>;
  /** Unary */
  setKeepSession(request: DeepPartial<SetKeepSessionRequest>, options?: CallOptions & CallOptionsExt): Promise<Respone>;
  deleteSession(request: DeepPartial<DeleteSessionRequest>, options?: CallOptions & CallOptionsExt): Promise<Respone>;
  setName(request: DeepPartial<SetNameRequest>, options?: CallOptions & CallOptionsExt): Promise<Respone>;
  createSegment(request: DeepPartial<CreateSegmentRequest>, options?: CallOptions & CallOptionsExt): Promise<Respone>;
  deleteSegment(request: DeepPartial<DeleteSegmentRequest>, options?: CallOptions & CallOptionsExt): Promise<Respone>;
  renderSegment(request: DeepPartial<RenderSegmentRequest>, options?: CallOptions & CallOptionsExt): Promise<Respone>;
  updateSegment(request: DeepPartial<UpdateSegmentRequest>, options?: CallOptions & CallOptionsExt): Promise<Respone>;
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

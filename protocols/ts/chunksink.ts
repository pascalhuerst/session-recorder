/* eslint-disable */
import Long from "long";
import type { CallContext, CallOptions } from "nice-grpc-common";
import _m0 from "protobufjs/minimal";

export const protobufPackage = "chunksink";

export enum AudioInputStatus {
  UNKNOWN = 0,
  NO_SIGNAL = 1,
  SIGNAL = 2,
  UNRECOGNIZED = -1,
}

export function audioInputStatusFromJSON(object: any): AudioInputStatus {
  switch (object) {
    case 0:
    case "UNKNOWN":
      return AudioInputStatus.UNKNOWN;
    case 1:
    case "NO_SIGNAL":
      return AudioInputStatus.NO_SIGNAL;
    case 2:
    case "SIGNAL":
      return AudioInputStatus.SIGNAL;
    case -1:
    case "UNRECOGNIZED":
    default:
      return AudioInputStatus.UNRECOGNIZED;
  }
}

export function audioInputStatusToJSON(object: AudioInputStatus): string {
  switch (object) {
    case AudioInputStatus.UNKNOWN:
      return "UNKNOWN";
    case AudioInputStatus.NO_SIGNAL:
      return "NO_SIGNAL";
    case AudioInputStatus.SIGNAL:
      return "SIGNAL";
    case AudioInputStatus.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface RecorderStatusRequest {
  name: string;
  uuid: string;
  status: AudioInputStatus;
}

export interface RecorderStatusReply {
  sendChunks: boolean;
}

export interface StreamChunkDataRequest {
}

export interface ChunkSourceMetrics {
  sessionID: string;
  chunkCount: number;
  startTime: number;
}

export interface ChunkData {
  name: string;
  uuid: string;
  metrics: ChunkSourceMetrics | undefined;
  samples: number[];
}

function createBaseRecorderStatusRequest(): RecorderStatusRequest {
  return { name: "", uuid: "", status: 0 };
}

export const RecorderStatusRequest = {
  encode(message: RecorderStatusRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.uuid !== "") {
      writer.uint32(18).string(message.uuid);
    }
    if (message.status !== 0) {
      writer.uint32(24).int32(message.status);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): RecorderStatusRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRecorderStatusRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.uuid = reader.string();
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.status = reader.int32() as any;
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): RecorderStatusRequest {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      uuid: isSet(object.uuid) ? globalThis.String(object.uuid) : "",
      status: isSet(object.status) ? audioInputStatusFromJSON(object.status) : 0,
    };
  },

  toJSON(message: RecorderStatusRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.uuid !== "") {
      obj.uuid = message.uuid;
    }
    if (message.status !== 0) {
      obj.status = audioInputStatusToJSON(message.status);
    }
    return obj;
  },

  create(base?: DeepPartial<RecorderStatusRequest>): RecorderStatusRequest {
    return RecorderStatusRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<RecorderStatusRequest>): RecorderStatusRequest {
    const message = createBaseRecorderStatusRequest();
    message.name = object.name ?? "";
    message.uuid = object.uuid ?? "";
    message.status = object.status ?? 0;
    return message;
  },
};

function createBaseRecorderStatusReply(): RecorderStatusReply {
  return { sendChunks: false };
}

export const RecorderStatusReply = {
  encode(message: RecorderStatusReply, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.sendChunks === true) {
      writer.uint32(8).bool(message.sendChunks);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): RecorderStatusReply {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRecorderStatusReply();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.sendChunks = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): RecorderStatusReply {
    return { sendChunks: isSet(object.sendChunks) ? globalThis.Boolean(object.sendChunks) : false };
  },

  toJSON(message: RecorderStatusReply): unknown {
    const obj: any = {};
    if (message.sendChunks === true) {
      obj.sendChunks = message.sendChunks;
    }
    return obj;
  },

  create(base?: DeepPartial<RecorderStatusReply>): RecorderStatusReply {
    return RecorderStatusReply.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<RecorderStatusReply>): RecorderStatusReply {
    const message = createBaseRecorderStatusReply();
    message.sendChunks = object.sendChunks ?? false;
    return message;
  },
};

function createBaseStreamChunkDataRequest(): StreamChunkDataRequest {
  return {};
}

export const StreamChunkDataRequest = {
  encode(_: StreamChunkDataRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): StreamChunkDataRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseStreamChunkDataRequest();
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

  fromJSON(_: any): StreamChunkDataRequest {
    return {};
  },

  toJSON(_: StreamChunkDataRequest): unknown {
    const obj: any = {};
    return obj;
  },

  create(base?: DeepPartial<StreamChunkDataRequest>): StreamChunkDataRequest {
    return StreamChunkDataRequest.fromPartial(base ?? {});
  },
  fromPartial(_: DeepPartial<StreamChunkDataRequest>): StreamChunkDataRequest {
    const message = createBaseStreamChunkDataRequest();
    return message;
  },
};

function createBaseChunkSourceMetrics(): ChunkSourceMetrics {
  return { sessionID: "", chunkCount: 0, startTime: 0 };
}

export const ChunkSourceMetrics = {
  encode(message: ChunkSourceMetrics, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.sessionID !== "") {
      writer.uint32(10).string(message.sessionID);
    }
    if (message.chunkCount !== 0) {
      writer.uint32(16).uint32(message.chunkCount);
    }
    if (message.startTime !== 0) {
      writer.uint32(24).uint64(message.startTime);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ChunkSourceMetrics {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseChunkSourceMetrics();
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

          message.chunkCount = reader.uint32();
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.startTime = longToNumber(reader.uint64() as Long);
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ChunkSourceMetrics {
    return {
      sessionID: isSet(object.sessionID) ? globalThis.String(object.sessionID) : "",
      chunkCount: isSet(object.chunkCount) ? globalThis.Number(object.chunkCount) : 0,
      startTime: isSet(object.startTime) ? globalThis.Number(object.startTime) : 0,
    };
  },

  toJSON(message: ChunkSourceMetrics): unknown {
    const obj: any = {};
    if (message.sessionID !== "") {
      obj.sessionID = message.sessionID;
    }
    if (message.chunkCount !== 0) {
      obj.chunkCount = Math.round(message.chunkCount);
    }
    if (message.startTime !== 0) {
      obj.startTime = Math.round(message.startTime);
    }
    return obj;
  },

  create(base?: DeepPartial<ChunkSourceMetrics>): ChunkSourceMetrics {
    return ChunkSourceMetrics.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ChunkSourceMetrics>): ChunkSourceMetrics {
    const message = createBaseChunkSourceMetrics();
    message.sessionID = object.sessionID ?? "";
    message.chunkCount = object.chunkCount ?? 0;
    message.startTime = object.startTime ?? 0;
    return message;
  },
};

function createBaseChunkData(): ChunkData {
  return { name: "", uuid: "", metrics: undefined, samples: [] };
}

export const ChunkData = {
  encode(message: ChunkData, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.uuid !== "") {
      writer.uint32(18).string(message.uuid);
    }
    if (message.metrics !== undefined) {
      ChunkSourceMetrics.encode(message.metrics, writer.uint32(26).fork()).ldelim();
    }
    writer.uint32(34).fork();
    for (const v of message.samples) {
      writer.uint32(v);
    }
    writer.ldelim();
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ChunkData {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseChunkData();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.uuid = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.metrics = ChunkSourceMetrics.decode(reader, reader.uint32());
          continue;
        case 4:
          if (tag === 32) {
            message.samples.push(reader.uint32());

            continue;
          }

          if (tag === 34) {
            const end2 = reader.uint32() + reader.pos;
            while (reader.pos < end2) {
              message.samples.push(reader.uint32());
            }

            continue;
          }

          break;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ChunkData {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      uuid: isSet(object.uuid) ? globalThis.String(object.uuid) : "",
      metrics: isSet(object.metrics) ? ChunkSourceMetrics.fromJSON(object.metrics) : undefined,
      samples: globalThis.Array.isArray(object?.samples) ? object.samples.map((e: any) => globalThis.Number(e)) : [],
    };
  },

  toJSON(message: ChunkData): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.uuid !== "") {
      obj.uuid = message.uuid;
    }
    if (message.metrics !== undefined) {
      obj.metrics = ChunkSourceMetrics.toJSON(message.metrics);
    }
    if (message.samples?.length) {
      obj.samples = message.samples.map((e) => Math.round(e));
    }
    return obj;
  },

  create(base?: DeepPartial<ChunkData>): ChunkData {
    return ChunkData.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ChunkData>): ChunkData {
    const message = createBaseChunkData();
    message.name = object.name ?? "";
    message.uuid = object.uuid ?? "";
    message.metrics = (object.metrics !== undefined && object.metrics !== null)
      ? ChunkSourceMetrics.fromPartial(object.metrics)
      : undefined;
    message.samples = object.samples?.map((e) => e) || [];
    return message;
  },
};

export type ChunkSinkDefinition = typeof ChunkSinkDefinition;
export const ChunkSinkDefinition = {
  name: "ChunkSink",
  fullName: "chunksink.ChunkSink",
  methods: {
    streamChunkData: {
      name: "StreamChunkData",
      requestType: StreamChunkDataRequest,
      requestStream: false,
      responseType: ChunkData,
      responseStream: true,
      options: {},
    },
    setRecorderStatus: {
      name: "SetRecorderStatus",
      requestType: RecorderStatusRequest,
      requestStream: false,
      responseType: RecorderStatusReply,
      responseStream: false,
      options: {},
    },
  },
} as const;

export interface ChunkSinkServiceImplementation<CallContextExt = {}> {
  streamChunkData(
    request: StreamChunkDataRequest,
    context: CallContext & CallContextExt,
  ): ServerStreamingMethodResult<DeepPartial<ChunkData>>;
  setRecorderStatus(
    request: RecorderStatusRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<RecorderStatusReply>>;
}

export interface ChunkSinkClient<CallOptionsExt = {}> {
  streamChunkData(
    request: DeepPartial<StreamChunkDataRequest>,
    options?: CallOptions & CallOptionsExt,
  ): AsyncIterable<ChunkData>;
  setRecorderStatus(
    request: DeepPartial<RecorderStatusRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<RecorderStatusReply>;
}

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends globalThis.Array<infer U> ? globalThis.Array<DeepPartial<U>>
  : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function longToNumber(long: Long): number {
  if (long.gt(globalThis.Number.MAX_SAFE_INTEGER)) {
    throw new globalThis.Error("Value is larger than Number.MAX_SAFE_INTEGER");
  }
  return long.toNumber();
}

if (_m0.util.Long !== Long) {
  _m0.util.Long = Long as any;
  _m0.configure();
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}

export type ServerStreamingMethodResult<Response> = { [Symbol.asyncIterator](): AsyncIterator<Response, void> };

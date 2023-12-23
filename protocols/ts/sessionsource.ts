/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";
import Long = require("long");

export const protobufPackage = "sessionsource";

export interface StreamOpenSessionsRequest {
}

export interface OpenSessions {
  openSessions: Session[];
}

export interface Session {
  ID: string;
  audioFileNames: string[];
  waveformDataFile: string;
  timeCreated: number;
  length: number;
  livetimeHours: number;
  keepSession: boolean;
}

function createBaseStreamOpenSessionsRequest(): StreamOpenSessionsRequest {
  return {};
}

export const StreamOpenSessionsRequest = {
  encode(_: StreamOpenSessionsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): StreamOpenSessionsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseStreamOpenSessionsRequest();
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

  fromJSON(_: any): StreamOpenSessionsRequest {
    return {};
  },

  toJSON(_: StreamOpenSessionsRequest): unknown {
    const obj: any = {};
    return obj;
  },

  create(base?: DeepPartial<StreamOpenSessionsRequest>): StreamOpenSessionsRequest {
    return StreamOpenSessionsRequest.fromPartial(base ?? {});
  },
  fromPartial(_: DeepPartial<StreamOpenSessionsRequest>): StreamOpenSessionsRequest {
    const message = createBaseStreamOpenSessionsRequest();
    return message;
  },
};

function createBaseOpenSessions(): OpenSessions {
  return { openSessions: [] };
}

export const OpenSessions = {
  encode(message: OpenSessions, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.openSessions) {
      Session.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): OpenSessions {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseOpenSessions();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.openSessions.push(Session.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): OpenSessions {
    return {
      openSessions: globalThis.Array.isArray(object?.openSessions)
        ? object.openSessions.map((e: any) => Session.fromJSON(e))
        : [],
    };
  },

  toJSON(message: OpenSessions): unknown {
    const obj: any = {};
    if (message.openSessions?.length) {
      obj.openSessions = message.openSessions.map((e) => Session.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<OpenSessions>): OpenSessions {
    return OpenSessions.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<OpenSessions>): OpenSessions {
    const message = createBaseOpenSessions();
    message.openSessions = object.openSessions?.map((e) => Session.fromPartial(e)) || [];
    return message;
  },
};

function createBaseSession(): Session {
  return {
    ID: "",
    audioFileNames: [],
    waveformDataFile: "",
    timeCreated: 0,
    length: 0,
    livetimeHours: 0,
    keepSession: false,
  };
}

export const Session = {
  encode(message: Session, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.ID !== "") {
      writer.uint32(10).string(message.ID);
    }
    for (const v of message.audioFileNames) {
      writer.uint32(18).string(v!);
    }
    if (message.waveformDataFile !== "") {
      writer.uint32(26).string(message.waveformDataFile);
    }
    if (message.timeCreated !== 0) {
      writer.uint32(32).int64(message.timeCreated);
    }
    if (message.length !== 0) {
      writer.uint32(40).int64(message.length);
    }
    if (message.livetimeHours !== 0) {
      writer.uint32(53).float(message.livetimeHours);
    }
    if (message.keepSession === true) {
      writer.uint32(56).bool(message.keepSession);
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

          message.audioFileNames.push(reader.string());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.waveformDataFile = reader.string();
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.timeCreated = longToNumber(reader.int64() as Long);
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.length = longToNumber(reader.int64() as Long);
          continue;
        case 6:
          if (tag !== 53) {
            break;
          }

          message.livetimeHours = reader.float();
          continue;
        case 7:
          if (tag !== 56) {
            break;
          }

          message.keepSession = reader.bool();
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
      audioFileNames: globalThis.Array.isArray(object?.audioFileNames)
        ? object.audioFileNames.map((e: any) => globalThis.String(e))
        : [],
      waveformDataFile: isSet(object.waveformDataFile) ? globalThis.String(object.waveformDataFile) : "",
      timeCreated: isSet(object.timeCreated) ? globalThis.Number(object.timeCreated) : 0,
      length: isSet(object.length) ? globalThis.Number(object.length) : 0,
      livetimeHours: isSet(object.livetimeHours) ? globalThis.Number(object.livetimeHours) : 0,
      keepSession: isSet(object.keepSession) ? globalThis.Boolean(object.keepSession) : false,
    };
  },

  toJSON(message: Session): unknown {
    const obj: any = {};
    if (message.ID !== "") {
      obj.ID = message.ID;
    }
    if (message.audioFileNames?.length) {
      obj.audioFileNames = message.audioFileNames;
    }
    if (message.waveformDataFile !== "") {
      obj.waveformDataFile = message.waveformDataFile;
    }
    if (message.timeCreated !== 0) {
      obj.timeCreated = Math.round(message.timeCreated);
    }
    if (message.length !== 0) {
      obj.length = Math.round(message.length);
    }
    if (message.livetimeHours !== 0) {
      obj.livetimeHours = message.livetimeHours;
    }
    if (message.keepSession === true) {
      obj.keepSession = message.keepSession;
    }
    return obj;
  },

  create(base?: DeepPartial<Session>): Session {
    return Session.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Session>): Session {
    const message = createBaseSession();
    message.ID = object.ID ?? "";
    message.audioFileNames = object.audioFileNames?.map((e) => e) || [];
    message.waveformDataFile = object.waveformDataFile ?? "";
    message.timeCreated = object.timeCreated ?? 0;
    message.length = object.length ?? 0;
    message.livetimeHours = object.livetimeHours ?? 0;
    message.keepSession = object.keepSession ?? false;
    return message;
  },
};

export type SessionSourceDefinition = typeof SessionSourceDefinition;
export const SessionSourceDefinition = {
  name: "SessionSource",
  fullName: "sessionsource.SessionSource",
  methods: {
    streamOpenSessions: {
      name: "StreamOpenSessions",
      requestType: StreamOpenSessionsRequest,
      requestStream: false,
      responseType: OpenSessions,
      responseStream: true,
      options: {},
    },
  },
} as const;

export interface SessionSourceServiceImplementation<CallContextExt = {}> {
  streamOpenSessions(
    request: StreamOpenSessionsRequest,
    context: CallContext & CallContextExt,
  ): ServerStreamingMethodResult<DeepPartial<OpenSessions>>;
}

export interface SessionSourceClient<CallOptionsExt = {}> {
  streamOpenSessions(
    request: DeepPartial<StreamOpenSessionsRequest>,
    options?: CallOptions & CallOptionsExt,
  ): AsyncIterable<OpenSessions>;
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

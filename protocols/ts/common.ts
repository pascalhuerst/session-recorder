/* eslint-disable */
import _m0 from "protobufjs/minimal";

export const protobufPackage = "common";

export enum SignalStatus {
  UNKNOWN = 0,
  NO_SIGNAL = 1,
  SIGNAL = 2,
  UNRECOGNIZED = -1,
}

export function signalStatusFromJSON(object: any): SignalStatus {
  switch (object) {
    case 0:
    case "UNKNOWN":
      return SignalStatus.UNKNOWN;
    case 1:
    case "NO_SIGNAL":
      return SignalStatus.NO_SIGNAL;
    case 2:
    case "SIGNAL":
      return SignalStatus.SIGNAL;
    case -1:
    case "UNRECOGNIZED":
    default:
      return SignalStatus.UNRECOGNIZED;
  }
}

export function signalStatusToJSON(object: SignalStatus): string {
  switch (object) {
    case SignalStatus.UNKNOWN:
      return "UNKNOWN";
    case SignalStatus.NO_SIGNAL:
      return "NO_SIGNAL";
    case SignalStatus.SIGNAL:
      return "SIGNAL";
    case SignalStatus.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface RecorderStatus {
  recorderID: string;
  recorderName: string;
  signalStatus: SignalStatus;
  rmsPercent: number;
  clipping: boolean;
}

export interface Respone {
  success: boolean;
  errorMessage: string;
}

function createBaseRecorderStatus(): RecorderStatus {
  return { recorderID: "", recorderName: "", signalStatus: 0, rmsPercent: 0, clipping: false };
}

export const RecorderStatus = {
  encode(message: RecorderStatus, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.recorderID !== "") {
      writer.uint32(10).string(message.recorderID);
    }
    if (message.recorderName !== "") {
      writer.uint32(18).string(message.recorderName);
    }
    if (message.signalStatus !== 0) {
      writer.uint32(24).int32(message.signalStatus);
    }
    if (message.rmsPercent !== 0) {
      writer.uint32(33).double(message.rmsPercent);
    }
    if (message.clipping === true) {
      writer.uint32(40).bool(message.clipping);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): RecorderStatus {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRecorderStatus();
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
          if (tag !== 24) {
            break;
          }

          message.signalStatus = reader.int32() as any;
          continue;
        case 4:
          if (tag !== 33) {
            break;
          }

          message.rmsPercent = reader.double();
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.clipping = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): RecorderStatus {
    return {
      recorderID: isSet(object.recorderID) ? globalThis.String(object.recorderID) : "",
      recorderName: isSet(object.recorderName) ? globalThis.String(object.recorderName) : "",
      signalStatus: isSet(object.signalStatus) ? signalStatusFromJSON(object.signalStatus) : 0,
      rmsPercent: isSet(object.rmsPercent) ? globalThis.Number(object.rmsPercent) : 0,
      clipping: isSet(object.clipping) ? globalThis.Boolean(object.clipping) : false,
    };
  },

  toJSON(message: RecorderStatus): unknown {
    const obj: any = {};
    if (message.recorderID !== "") {
      obj.recorderID = message.recorderID;
    }
    if (message.recorderName !== "") {
      obj.recorderName = message.recorderName;
    }
    if (message.signalStatus !== 0) {
      obj.signalStatus = signalStatusToJSON(message.signalStatus);
    }
    if (message.rmsPercent !== 0) {
      obj.rmsPercent = message.rmsPercent;
    }
    if (message.clipping === true) {
      obj.clipping = message.clipping;
    }
    return obj;
  },

  create(base?: DeepPartial<RecorderStatus>): RecorderStatus {
    return RecorderStatus.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<RecorderStatus>): RecorderStatus {
    const message = createBaseRecorderStatus();
    message.recorderID = object.recorderID ?? "";
    message.recorderName = object.recorderName ?? "";
    message.signalStatus = object.signalStatus ?? 0;
    message.rmsPercent = object.rmsPercent ?? 0;
    message.clipping = object.clipping ?? false;
    return message;
  },
};

function createBaseRespone(): Respone {
  return { success: false, errorMessage: "" };
}

export const Respone = {
  encode(message: Respone, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.success === true) {
      writer.uint32(8).bool(message.success);
    }
    if (message.errorMessage !== "") {
      writer.uint32(18).string(message.errorMessage);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Respone {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRespone();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.success = reader.bool();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.errorMessage = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Respone {
    return {
      success: isSet(object.success) ? globalThis.Boolean(object.success) : false,
      errorMessage: isSet(object.errorMessage) ? globalThis.String(object.errorMessage) : "",
    };
  },

  toJSON(message: Respone): unknown {
    const obj: any = {};
    if (message.success === true) {
      obj.success = message.success;
    }
    if (message.errorMessage !== "") {
      obj.errorMessage = message.errorMessage;
    }
    return obj;
  },

  create(base?: DeepPartial<Respone>): Respone {
    return Respone.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Respone>): Respone {
    const message = createBaseRespone();
    message.success = object.success ?? false;
    message.errorMessage = object.errorMessage ?? "";
    return message;
  },
};

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends globalThis.Array<infer U> ? globalThis.Array<DeepPartial<U>>
  : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}

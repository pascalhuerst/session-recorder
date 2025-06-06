// @generated by protobuf-ts 2.11.0 with parameter generate_dependencies,long_type_string,client_generic
// @generated from protobuf file "common.proto" (package "common", syntax proto3)
// tslint:disable
import type { BinaryWriteOptions } from "@protobuf-ts/runtime";
import type { IBinaryWriter } from "@protobuf-ts/runtime";
import { WireType } from "@protobuf-ts/runtime";
import type { BinaryReadOptions } from "@protobuf-ts/runtime";
import type { IBinaryReader } from "@protobuf-ts/runtime";
import { UnknownFieldHandler } from "@protobuf-ts/runtime";
import type { PartialMessage } from "@protobuf-ts/runtime";
import { reflectionMergePartial } from "@protobuf-ts/runtime";
import { MessageType } from "@protobuf-ts/runtime";
/**
 * @generated from protobuf message common.RecorderStatus
 */
export interface RecorderStatus {
    /**
     * @generated from protobuf field: string recorderID = 1
     */
    recorderID: string;
    /**
     * @generated from protobuf field: string recorderName = 2
     */
    recorderName: string;
    /**
     * @generated from protobuf field: common.SignalStatus signalStatus = 3
     */
    signalStatus: SignalStatus;
    /**
     * @generated from protobuf field: double rmsPercent = 4
     */
    rmsPercent: number;
    /**
     * @generated from protobuf field: bool clipping = 5
     */
    clipping: boolean;
}
/**
 * @generated from protobuf message common.Respone
 */
export interface Respone {
    /**
     * @generated from protobuf field: bool success = 1
     */
    success: boolean;
    /**
     * @generated from protobuf field: string errorMessage = 2
     */
    errorMessage: string;
}
/**
 * @generated from protobuf enum common.SignalStatus
 */
export enum SignalStatus {
    /**
     * @generated from protobuf enum value: UNKNOWN = 0;
     */
    UNKNOWN = 0,
    /**
     * @generated from protobuf enum value: NO_SIGNAL = 1;
     */
    NO_SIGNAL = 1,
    /**
     * @generated from protobuf enum value: SIGNAL = 2;
     */
    SIGNAL = 2
}
// @generated message type with reflection information, may provide speed optimized methods
class RecorderStatus$Type extends MessageType<RecorderStatus> {
    constructor() {
        super("common.RecorderStatus", [
            { no: 1, name: "recorderID", kind: "scalar", T: 9 /*ScalarType.STRING*/ },
            { no: 2, name: "recorderName", kind: "scalar", T: 9 /*ScalarType.STRING*/ },
            { no: 3, name: "signalStatus", kind: "enum", T: () => ["common.SignalStatus", SignalStatus] },
            { no: 4, name: "rmsPercent", kind: "scalar", T: 1 /*ScalarType.DOUBLE*/ },
            { no: 5, name: "clipping", kind: "scalar", T: 8 /*ScalarType.BOOL*/ }
        ]);
    }
    create(value?: PartialMessage<RecorderStatus>): RecorderStatus {
        const message = globalThis.Object.create((this.messagePrototype!));
        message.recorderID = "";
        message.recorderName = "";
        message.signalStatus = 0;
        message.rmsPercent = 0;
        message.clipping = false;
        if (value !== undefined)
            reflectionMergePartial<RecorderStatus>(this, message, value);
        return message;
    }
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: RecorderStatus): RecorderStatus {
        let message = target ?? this.create(), end = reader.pos + length;
        while (reader.pos < end) {
            let [fieldNo, wireType] = reader.tag();
            switch (fieldNo) {
                case /* string recorderID */ 1:
                    message.recorderID = reader.string();
                    break;
                case /* string recorderName */ 2:
                    message.recorderName = reader.string();
                    break;
                case /* common.SignalStatus signalStatus */ 3:
                    message.signalStatus = reader.int32();
                    break;
                case /* double rmsPercent */ 4:
                    message.rmsPercent = reader.double();
                    break;
                case /* bool clipping */ 5:
                    message.clipping = reader.bool();
                    break;
                default:
                    let u = options.readUnknownField;
                    if (u === "throw")
                        throw new globalThis.Error(`Unknown field ${fieldNo} (wire type ${wireType}) for ${this.typeName}`);
                    let d = reader.skip(wireType);
                    if (u !== false)
                        (u === true ? UnknownFieldHandler.onRead : u)(this.typeName, message, fieldNo, wireType, d);
            }
        }
        return message;
    }
    internalBinaryWrite(message: RecorderStatus, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter {
        /* string recorderID = 1; */
        if (message.recorderID !== "")
            writer.tag(1, WireType.LengthDelimited).string(message.recorderID);
        /* string recorderName = 2; */
        if (message.recorderName !== "")
            writer.tag(2, WireType.LengthDelimited).string(message.recorderName);
        /* common.SignalStatus signalStatus = 3; */
        if (message.signalStatus !== 0)
            writer.tag(3, WireType.Varint).int32(message.signalStatus);
        /* double rmsPercent = 4; */
        if (message.rmsPercent !== 0)
            writer.tag(4, WireType.Bit64).double(message.rmsPercent);
        /* bool clipping = 5; */
        if (message.clipping !== false)
            writer.tag(5, WireType.Varint).bool(message.clipping);
        let u = options.writeUnknownFields;
        if (u !== false)
            (u == true ? UnknownFieldHandler.onWrite : u)(this.typeName, message, writer);
        return writer;
    }
}
/**
 * @generated MessageType for protobuf message common.RecorderStatus
 */
export const RecorderStatus = new RecorderStatus$Type();
// @generated message type with reflection information, may provide speed optimized methods
class Respone$Type extends MessageType<Respone> {
    constructor() {
        super("common.Respone", [
            { no: 1, name: "success", kind: "scalar", T: 8 /*ScalarType.BOOL*/ },
            { no: 2, name: "errorMessage", kind: "scalar", T: 9 /*ScalarType.STRING*/ }
        ]);
    }
    create(value?: PartialMessage<Respone>): Respone {
        const message = globalThis.Object.create((this.messagePrototype!));
        message.success = false;
        message.errorMessage = "";
        if (value !== undefined)
            reflectionMergePartial<Respone>(this, message, value);
        return message;
    }
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: Respone): Respone {
        let message = target ?? this.create(), end = reader.pos + length;
        while (reader.pos < end) {
            let [fieldNo, wireType] = reader.tag();
            switch (fieldNo) {
                case /* bool success */ 1:
                    message.success = reader.bool();
                    break;
                case /* string errorMessage */ 2:
                    message.errorMessage = reader.string();
                    break;
                default:
                    let u = options.readUnknownField;
                    if (u === "throw")
                        throw new globalThis.Error(`Unknown field ${fieldNo} (wire type ${wireType}) for ${this.typeName}`);
                    let d = reader.skip(wireType);
                    if (u !== false)
                        (u === true ? UnknownFieldHandler.onRead : u)(this.typeName, message, fieldNo, wireType, d);
            }
        }
        return message;
    }
    internalBinaryWrite(message: Respone, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter {
        /* bool success = 1; */
        if (message.success !== false)
            writer.tag(1, WireType.Varint).bool(message.success);
        /* string errorMessage = 2; */
        if (message.errorMessage !== "")
            writer.tag(2, WireType.LengthDelimited).string(message.errorMessage);
        let u = options.writeUnknownFields;
        if (u !== false)
            (u == true ? UnknownFieldHandler.onWrite : u)(this.typeName, message, writer);
        return writer;
    }
}
/**
 * @generated MessageType for protobuf message common.Respone
 */
export const Respone = new Respone$Type();

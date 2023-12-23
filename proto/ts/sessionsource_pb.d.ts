// package: sessionsource
// file: sessionsource.proto

import * as jspb from "google-protobuf";

export class StreamOpenSessionsRequest extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StreamOpenSessionsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: StreamOpenSessionsRequest): StreamOpenSessionsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: StreamOpenSessionsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StreamOpenSessionsRequest;
  static deserializeBinaryFromReader(message: StreamOpenSessionsRequest, reader: jspb.BinaryReader): StreamOpenSessionsRequest;
}

export namespace StreamOpenSessionsRequest {
  export type AsObject = {
  }
}

export class OpenSessions extends jspb.Message {
  clearOpensessionsList(): void;
  getOpensessionsList(): Array<Session>;
  setOpensessionsList(value: Array<Session>): void;
  addOpensessions(value?: Session, index?: number): Session;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): OpenSessions.AsObject;
  static toObject(includeInstance: boolean, msg: OpenSessions): OpenSessions.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: OpenSessions, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): OpenSessions;
  static deserializeBinaryFromReader(message: OpenSessions, reader: jspb.BinaryReader): OpenSessions;
}

export namespace OpenSessions {
  export type AsObject = {
    opensessionsList: Array<Session.AsObject>,
  }
}

export class Session extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  clearAudiofilenamesList(): void;
  getAudiofilenamesList(): Array<string>;
  setAudiofilenamesList(value: Array<string>): void;
  addAudiofilenames(value: string, index?: number): string;

  getWaveformdatafile(): string;
  setWaveformdatafile(value: string): void;

  getTimecreated(): number;
  setTimecreated(value: number): void;

  getLength(): number;
  setLength(value: number): void;

  getLivetimehours(): number;
  setLivetimehours(value: number): void;

  getKeepsession(): boolean;
  setKeepsession(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Session.AsObject;
  static toObject(includeInstance: boolean, msg: Session): Session.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Session, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Session;
  static deserializeBinaryFromReader(message: Session, reader: jspb.BinaryReader): Session;
}

export namespace Session {
  export type AsObject = {
    id: string,
    audiofilenamesList: Array<string>,
    waveformdatafile: string,
    timecreated: number,
    length: number,
    livetimehours: number,
    keepsession: boolean,
  }
}


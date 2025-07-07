export interface Segment {
  id: string;
  timeStart: Date;
  timeEnd: Date;
  name: string;
}

export interface SessionInfo_Files {
  ogg: string;
  flac: string;
  waveform: string;
}

export type Session = {
  id: string;
  startedAt: Date;
  finishedAt: Date;
  expiresAt: Date;
  name: string;
  keep: boolean;
  segments: Segment[];
  inlineFiles: SessionInfo_Files;
  downloadFiles: SessionInfo_Files;
};

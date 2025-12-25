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

export type SessionState = 'recording' | 'processing' | 'finished' | 'error';

export type Session = {
  id: string;
  startedAt: Date;
  finishedAt: Date | null;
  expiresAt: Date | null;
  name: string;
  keep: boolean;
  state: SessionState;
  errorMessage?: string;
  segments: Segment[];
  inlineFiles: SessionInfo_Files | null;
  downloadFiles: SessionInfo_Files | null;
};

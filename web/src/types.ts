export type SegmentState = 'unknown' | 'queued' | 'rendering' | 'finished' | 'error';

export interface SegmentFiles {
  ogg: string;
  flac: string;
}

export interface Segment {
  id: string;
  timeStart: Date;
  timeEnd: Date;
  name: string;
  state: SegmentState;
  errorMessage?: string;
  inlineFiles: SegmentFiles | null;
  downloadFiles: SegmentFiles | null;
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
  duration: number | null; // seconds (estimated during processing, exact after finished)
  name: string;
  keep: boolean;
  state: SessionState;
  errorMessage?: string;
  segments: Segment[];
  inlineFiles: SessionInfo_Files | null;
  downloadFiles: SessionInfo_Files | null;
};

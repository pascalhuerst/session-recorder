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
  name: string;
  keep: boolean;
  state: SessionState;
  errorMessage?: string;
  segments: Segment[];
  inlineFiles: SessionInfo_Files | null;
  downloadFiles: SessionInfo_Files | null;
};

export type ShowState = 'draft' | 'live' | 'ended' | 'archived';

export interface Act {
  id: string;
  name: string;
  plannedStart: Date;
  plannedEnd: Date;
  emails: string[];
  segmentId: string | null;
  actualStart: Date | null;
  actualEnd: Date | null;
}

export interface Show {
  id: string;
  name: string;
  date: Date;
  state: ShowState;
  recorderId: string;
  sessionId: string | null;
  acts: Act[];
}

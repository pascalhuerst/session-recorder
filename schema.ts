export type UUID = string

export enum SessionState {
    Recording = 'Recording',
    Closed = 'Closed',
    Processing = 'Processing'
}

export type File = {
    bucket: string
    key: string
    sampleFormat: SampleFormat
}

export type Session = {
    uuid: UUID
    device: Device
    startTime: Date
    endTime: Date
    state: SessionState
    lifetime: Date | null
    file: File
}

export type Device = {
    uud: UUID
    alias: string
}

export enum SampleFormatType {
    S16LE = 'S16LE'
}

export type SampleFormat = {
    uid: string
    rate: number
    format: SampleFormatType
}

export type SegmentMarker = {
    timestamp: Date
    session: Session
}

export type Segment = {
    start: SegmentMarker
    end: SegmentMarker
    metadata: Record<string, any> // raw id3 v2 tags
}

export type Recording = {
    file: File
    metadata: Segment['metadata'] // enhanced id3 v2 tags
    fileFormat: string
}

export type EmbedLink = {

}

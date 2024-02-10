export const overviewTheme = {
  enablePoints: false,
  enableSegments: true,
  playheadPadding: 0,
  playheadColor: 'red',
  playedWaveformColor: '#6b46c1',
  showPlayheadTime: true,
  playheadTextColor: '#6b46c1',
  playheadBackgroundColor: 'rgb(107,70,193,0.1)',
  waveformColor: '#767c89',
};

export type OverviewTheme = typeof overviewTheme;

export const zoomviewTheme = {
  enablePoints: false,
  enableSegments: true,
  playheadPadding: 16,
  playheadColor: '#6b46c1',
  waveformColor: '#a5aab4',
  showPlayheadTime: true,
  playedWaveformColor: '#bdafe3',
  playheadTextColor: '#6b46c1',
  segmentOptions: {
    startMarkerColor: '#d53f8c',
    endMarkerColor: '#d53f8c',
  },
};

export type ZoomviewTheme = typeof zoomviewTheme;

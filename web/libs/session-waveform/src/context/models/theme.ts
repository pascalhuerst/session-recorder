import { z } from 'zod';

export const overviewThemeSchema = z
  .object({
    enablePoints: z.boolean().default(false),
    enableSegments: z.boolean().default(true),
    playheadPadding: z.number().default(0),
    playheadColor: z.string().default('red'),
    playedWaveformColor: z.string().default('#6b46c1'),
    showPlayheadTime: z.boolean().default(true),
    playheadTextColor: z.string().default('#6b46c1'),
    playheadBackgroundColor: z.string().default('rgb(107,70,193,0.1)'),
    waveformColor: z.string().default('#767c89'),
  })
  .default({});

export const zoomviewThemeSchema = z
  .object({
    enablePoints: z.boolean().default(false),
    enableSegments: z.boolean().default(true),
    playheadPadding: z.number().default(16),
    playheadColor: z.string().default('#6b46c1'),
    waveformColor: z.string().default('#a5aab4'),
    showPlayheadTime: z.boolean().default(true),
    playedWaveformColor: z.string().default('#bdafe3'),
    playheadTextColor: z.string().default('#6b46c1'),
    segmentOptions: z
      .object({
        startMarkerColor: z.string().default('#d53f8c'),
        endMarkerColor: z.string().default('#d53f8c'),
      })
      .default({}),
  })
  .default({});

export const themeSchema = z
  .object({
    overviewTheme: overviewThemeSchema,
    zoomviewTheme: zoomviewThemeSchema,
  })
  .default({});

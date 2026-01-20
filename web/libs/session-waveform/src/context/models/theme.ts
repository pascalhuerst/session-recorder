import { z } from 'zod';

export const overviewThemeSchema = z
  .object({
    enablePoints: z.boolean().default(false),
    enableSegments: z.boolean().default(true),
    playheadPadding: z.number().default(0),
    playheadColor: z.string().default('#f43f5e'),
    playedWaveformColor: z.string().default('#14b8a6'),
    showPlayheadTime: z.boolean().default(true),
    playheadTextColor: z.string().default('#14b8a6'),
    playheadBackgroundColor: z.string().default('rgba(20,184,166,0.15)'),
    waveformColor: z.string().default('#94a3b8'),
  })
  .default({});

export const zoomviewThemeSchema = z
  .object({
    enablePoints: z.boolean().default(false),
    enableSegments: z.boolean().default(true),
    playheadPadding: z.number().default(16),
    playheadColor: z.string().default('#14b8a6'),
    waveformColor: z.string().default('#94a3b8'),
    showPlayheadTime: z.boolean().default(true),
    playedWaveformColor: z.string().default('#5eead4'),
    playheadTextColor: z.string().default('#14b8a6'),
    segmentOptions: z
      .object({
        startMarkerColor: z.string().default('#f43f5e'),
        endMarkerColor: z.string().default('#f43f5e'),
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

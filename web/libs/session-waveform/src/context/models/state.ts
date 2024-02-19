import { z } from 'zod';
import { themeSchema } from './theme';

export const audioUrlSchema = z.object({
  type: z.string(),
  src: z.string(),
});

export const permissionsSchema = z.object({
  create: z.boolean(),
  update: z.boolean(),
  delete: z.boolean(),
});

export const segmentSchema = z.object({
  id: z.string(),
  startTime: z.number(),
  endTime: z.number(),
  editable: z.boolean().default(true),
  color: z.string().optional(),
  labelText: z.string().optional(),
  startIndex: z.string(),
  endIndex: z.string(),
  deleted: z.boolean().optional(),
});

export const peaksModuleSchema = z.object({
  theme: themeSchema,
  audioUrls: z.array(audioUrlSchema).nonempty(),
  waveformUrl: z.string().url().optional(),
  player: z
    .object({
      isPlaying: z.boolean().default(false),
      duration: z.number().default(0),
      currentTime: z.number().default(0),
    })
    .default({}),
  zoom: z
    .object({
      zoomLevel: z.number().min(1).default(300),
      zoomStep: z.number().min(1).default(60),
    })
    .default({}),
  amplitude: z
    .object({
      amplitudeScale: z.number().min(0.1).default(0.6),
      amplitudeStep: z.number().min(0.1).default(0.1),
    })
    .default({}),
  permissions: permissionsSchema,
  segments: z.array(segmentSchema),
});

export type AudioSourceUrl = z.output<typeof audioUrlSchema>;
export type Permissions = z.output<typeof permissionsSchema>;
export type Segment = z.output<typeof segmentSchema>;

export type PeaksModuleState = z.output<typeof peaksModuleSchema>;

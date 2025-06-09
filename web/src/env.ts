import { z } from 'zod';

export const envSchema = z.object({
  PROD: z.boolean().default(false),
  // VITE_SERVER_URL: z.string().url(),
  VITE_GRPC_SERVER_URL: z.string().url(),
  VITE_FILE_SERVER_URL: z.string().url(),
});

export type EnvSchema = z.infer<typeof envSchema>;

export const envFactory = () => {
  return envSchema.parse(
    withOverrides({
      PROD: import.meta.env.PROD,
      // VITE_SERVER_URL: import.meta.env.VITE_SERVER_URL,
      VITE_GRPC_SERVER_URL: import.meta.env.VITE_GRPC_SERVER_URL,
      VITE_FILE_SERVER_URL: import.meta.env.VITE_FILE_SERVER_URL,
    }),
  );
};

export const withOverrides = (values: EnvSchema) => {
  return Object.entries(values).reduce((acc, [key, value]) => {
    const override = localStorage.getItem(key);
    return {
      ...acc,
      [key]: override !== null ? override : value,
    };
  }, {});
};

export const env = envFactory();

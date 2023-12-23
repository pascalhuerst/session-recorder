import { env } from "../env.ts";

export type OpenSession = {
  id: string
  wav_file_name: string
  ogg_file_name: string
  waveform_file_name: string
  timestamp: string
  hours_to_live: number
  flagged?: boolean
}


export const getOpenSessions = async (deviceId: string): Promise<OpenSession[]> => {
  return fetch(new URL("introspect", env.VITE_SERVER_URL))
    .then((response) => response.json())
    .then((data) => {
      return data[deviceId].open_sessions as Array<OpenSession>;
    })
    .then((data) => {
      return data.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());
    });
};
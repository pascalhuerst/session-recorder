import { env } from "../env.ts";

export const getRecordingDevices = async () => {
  return fetch(new URL("introspect", env.VITE_SERVER_URL))
    .then((response) => response.json())
    .then((data) => {
      return Object.keys(data);
    });
};
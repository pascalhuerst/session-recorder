export type Segment = {
  id: string;
  startTime: number;
  endTime: number;
  editable?: boolean;
  color?: string;
  labelText?: string;
  startIndex: string;
  endIndex: string;
  [key: string]: any;
};

export type AudioSourceUrl = { src: string; type: string };

export type Permissions = {
  create: boolean;
  update: boolean;
  delete: boolean;
};

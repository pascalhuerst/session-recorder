// Pollen.css color palette for segments (500 variants)
// These match --color-{name}-500 from pollen.css
// Note: purple excluded as it clashes with primary/play color
export const SEGMENT_COLORS_SOLID = [
  '#4299e1', // blue-500
  '#38b2ac', // teal-500
  '#48bb78', // green-500
  '#ed8936', // orange-500
  '#ed64a6', // pink-500
  '#ecc94b', // yellow-500
  '#795548', // brown-500
  '#e53e3e', // red-500
];

// Segment color palette with 40% transparency for waveform overlays
export const SEGMENT_COLORS = SEGMENT_COLORS_SOLID.map(
  (color) => `${color}66` // 40% opacity in hex
);

export const getSegmentColor = (index: number) => {
  return SEGMENT_COLORS[index % SEGMENT_COLORS.length];
};

export const getSegmentColorSolid = (index: number) => {
  return SEGMENT_COLORS_SOLID[index % SEGMENT_COLORS_SOLID.length];
};

// Convert color with alpha to solid color (strips alpha)
export const toSolidColor = (color: string): string => {
  // Handle hex with alpha (#RRGGBBAA)
  if (color.startsWith('#') && color.length === 9) {
    return color.slice(0, 7);
  }
  // Handle rgba
  const rgbaMatch = color.match(/rgba?\((\d+),\s*(\d+),\s*(\d+)/);
  if (rgbaMatch) {
    return `rgb(${rgbaMatch[1]}, ${rgbaMatch[2]}, ${rgbaMatch[3]})`;
  }
  return color;
};

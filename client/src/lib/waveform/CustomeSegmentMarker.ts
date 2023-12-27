import { CreateSegmentMarkerOptions } from "peaks.js";
import Konva from "konva";

export class CustomSegmentMarker {
  private _options: CreateSegmentMarkerOptions;
  private _handle?: Konva.Rect;
  private _index?: Konva.Text;
  private _line?: Konva.Line;

  constructor(options: CreateSegmentMarkerOptions) {
    this._options = options;
    console.log(options);
  }

  init(group: Konva.Group) {
    const layer = this._options.layer;
    const height = layer.getHeight();

    const isStart = this._options.startMarker;
    const x = isStart ? 0 : -24;
    const y = layer.getHeight() - 60;

    this._handle = new Konva.Rect({
      x: x,
      y: y,
      width: 24,
      height: 24,
      cornerRadius: isStart ? [0, 4, 4, 0] : [4, 0, 0, 4],
      fill: this._options.color as string
    });

    this._index = new Konva.Text({
      x: x + 7,
      y: y + 4,
      fontSize: 16,
      fontStyle: 'bold',
      fontFamily: 'Gabarito',
      text: isStart ? String(this._options.segment.startIndex) : String(this._options.segment.endIndex),
      fill: "white"
    });

    this._line = new Konva.Line({
      points: [0.5, 0, 0.5, height], // x1, y1, x2, y2
      stroke: this._options.color as string,
      strokeWidth: 1
    });

    group.add(this._handle);
    group.add(this._index);
    group.add(this._line);

    this._handle.on("mouseenter", () => {
      const highlightColor = "#ff0000";
      this._handle?.fill(highlightColor);
      this._line?.stroke(highlightColor);
      layer.draw();
    });

    this._handle.on("mouseleave", () => {
      const defaultColor = this._options.color as string;
      this._handle?.fill(defaultColor);
      this._line?.stroke(defaultColor);
      layer.draw();
    });
  }

  fitToView() {
    const layer = this._options.layer;
    const height = layer.getHeight();

    this._line?.points([0.5, 0, 0.5, height]);
  }

  update(options: any) {
    // For a point marker:
    if (options.time !== undefined) {
      console.log("Updated point marker time", options.time);
    }

    // For a segment start/end marker:
    if (options.startTime !== undefined && this._options.startMarker) {
      console.log("Updated segment start marker time", options.startTime);
    }

    if (options.endTime !== undefined && !this._options.startMarker) {
      console.log("Updated segment end marker time", options.endTime);
    }

    if (options.labelText !== undefined) {
      console.log("Updated label text", options.labelText);
    }

    if (options.color !== undefined) {
      this._line?.stroke(options.color);
    }

    if (options.editable !== undefined) {
      // Show or hide the Konva shapes that draw the marker
      console.log("Updated editable state", options.editable);
    }
  }

  destroy() {
    console.log("Marker destroyed");
  }
}
import { type CreateSegmentMarkerOptions } from 'peaks.js';
import Konva from 'konva';
import type { createEventEmitter } from '../lib/app/createEventEmitter';
import { toSolidColor } from '../lib/utils/segmentColors';

type Services = {
  eventEmitter: ReturnType<typeof createEventEmitter>;
};

export class CustomSegmentMarker {
  private _options: CreateSegmentMarkerOptions;
  private _services: Services;
  private _handle?: Konva.Rect;
  private _index?: Konva.Text;
  private _line?: Konva.Line;
  private _solidColor: string;

  constructor(options: CreateSegmentMarkerOptions, services: Services) {
    this._options = options;
    this._services = services;
    // Use segment.color as the source of truth (our custom property)
    // Fall back to options.color if segment.color is not available
    const segmentColor = (options.segment as any).color ?? options.color;
    this._solidColor = toSolidColor(segmentColor as string);
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
      fill: this._solidColor,
    });

    this._index = new Konva.Text({
      x: x + 7,
      y: y + 4,
      fontSize: 16,
      fontStyle: 'bold',
      fontFamily: 'Gabarito',
      text: isStart
        ? String(this._options.segment.startIndex)
        : String(this._options.segment.endIndex),
      fill: 'white',
    });

    this._line = new Konva.Line({
      points: [0.5, 0, 0.5, height], // x1, y1, x2, y2
      stroke: this._solidColor,
      strokeWidth: 1,
    });

    group.add(this._handle);
    group.add(this._index);
    group.add(this._line);

    this._handle.on('mouseenter', () => {
      const highlightColor = '#ff0000';
      this._handle?.fill(highlightColor);
      this._line?.stroke(highlightColor);
      layer.draw();
    });

    this._handle.on('mouseleave', () => {
      this._handle?.fill(this._solidColor);
      this._line?.stroke(this._solidColor);
      layer.draw();
    });
  }

  fitToView() {
    const layer = this._options.layer;
    const height = layer.getHeight();

    this._line?.points([0.5, 0, 0.5, height]);
  }

  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  update(_options: unknown) {
    // Note: segmentUpdated is emitted by the updateSegment command handler
    // in installSegmentsControls.ts for programmatic updates, and by the
    // segments.dragend handler for drag updates. We don't emit here to avoid
    // duplicate events during drag (which calls update() on every mouse move).
  }

  destroy() {
    // Note: segmentRemoved is emitted by the segments.remove event handler
    // in installSegmentsControls.ts, so we don't emit it here
  }
}

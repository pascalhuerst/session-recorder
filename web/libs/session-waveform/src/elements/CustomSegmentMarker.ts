import { type CreateSegmentMarkerOptions } from 'peaks.js';
import Konva from 'konva';
import type { createEventEmitter } from '../lib/app/createEventEmitter';

type Services = {
  eventEmitter: ReturnType<typeof createEventEmitter>;
};

export class CustomSegmentMarker {
  private _options: CreateSegmentMarkerOptions;
  private _services: Services;
  private _handle?: Konva.Rect;
  private _index?: Konva.Text;
  private _line?: Konva.Line;

  constructor(options: CreateSegmentMarkerOptions, services: Services) {
    this._options = options;
    this._services = services;
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
      fill: this._options.color as string,
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
      stroke: this._options.color as string,
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
    if (this._options.segment.id) {
      this._services.eventEmitter.emit(
        'segmentUpdated',
        this._options.segment.id,
        options
      );
    }
  }

  destroy() {
    if (this._options.segment.id) {
      this._services.eventEmitter.emit(
        'segmentRemoved',
        this._options.segment.id
      );
    }
  }
}

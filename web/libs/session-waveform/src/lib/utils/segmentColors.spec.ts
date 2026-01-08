/**
 * Test Plan: segmentColors
 *
 * Scenario: Get segment color by index
 *   Given a segment index
 *   When getSegmentColor is called
 *   Then the corresponding color with 40% opacity should be returned
 *   And indexes should wrap around when exceeding array length
 *
 * Scenario: Get solid segment color by index
 *   Given a segment index
 *   When getSegmentColorSolid is called
 *   Then the corresponding solid color should be returned
 *   And indexes should wrap around when exceeding array length
 *
 * Scenario: Convert hex with alpha to solid color
 *   Given a hex color with alpha (#RRGGBBAA)
 *   When toSolidColor is called
 *   Then the alpha should be stripped (#RRGGBB)
 *
 * Scenario: Convert rgba to solid rgb
 *   Given an rgba color string
 *   When toSolidColor is called
 *   Then it should return rgb without alpha
 *
 * Scenario: Pass through already solid colors
 *   Given a color without alpha
 *   When toSolidColor is called
 *   Then the color should be returned unchanged
 */

import { describe, it, expect } from 'vitest';
import {
  SEGMENT_COLORS,
  SEGMENT_COLORS_SOLID,
  getSegmentColor,
  getSegmentColorSolid,
  toSolidColor,
} from './segmentColors';

describe('segmentColors', () => {
  describe('SEGMENT_COLORS_SOLID', () => {
    it('should have 8 solid colors', () => {
      expect(SEGMENT_COLORS_SOLID).toHaveLength(8);
    });

    it('should contain valid hex colors', () => {
      SEGMENT_COLORS_SOLID.forEach((color) => {
        expect(color).toMatch(/^#[0-9a-fA-F]{6}$/);
      });
    });

    it('should not include purple (primary color)', () => {
      // Purple variants typically start with #8 or #9 or contain 'a855f7' etc.
      const purplePatterns = [
        '#8b5cf6',
        '#a855f7',
        '#9333ea',
        '#7c3aed',
        '#6d28d9',
      ];
      SEGMENT_COLORS_SOLID.forEach((color) => {
        expect(purplePatterns).not.toContain(color.toLowerCase());
      });
    });
  });

  describe('SEGMENT_COLORS', () => {
    it('should have same length as SEGMENT_COLORS_SOLID', () => {
      expect(SEGMENT_COLORS).toHaveLength(SEGMENT_COLORS_SOLID.length);
    });

    it('should have colors with 40% opacity (66 hex suffix)', () => {
      SEGMENT_COLORS.forEach((color) => {
        expect(color).toMatch(/^#[0-9a-fA-F]{6}66$/);
        expect(color).toHaveLength(9);
      });
    });

    it('should derive from SEGMENT_COLORS_SOLID with alpha suffix', () => {
      SEGMENT_COLORS.forEach((color, index) => {
        expect(color).toBe(SEGMENT_COLORS_SOLID[index] + '66');
      });
    });
  });

  describe('getSegmentColor', () => {
    it('should return first color for index 0', () => {
      expect(getSegmentColor(0)).toBe(SEGMENT_COLORS[0]);
    });

    it('should return correct color for valid indexes', () => {
      for (let i = 0; i < SEGMENT_COLORS.length; i++) {
        expect(getSegmentColor(i)).toBe(SEGMENT_COLORS[i]);
      }
    });

    it('should wrap around for indexes exceeding array length', () => {
      expect(getSegmentColor(8)).toBe(SEGMENT_COLORS[0]);
      expect(getSegmentColor(9)).toBe(SEGMENT_COLORS[1]);
      expect(getSegmentColor(16)).toBe(SEGMENT_COLORS[0]);
    });

    it('should handle large indexes', () => {
      expect(getSegmentColor(100)).toBe(SEGMENT_COLORS[100 % 8]);
      expect(getSegmentColor(1000)).toBe(SEGMENT_COLORS[1000 % 8]);
    });
  });

  describe('getSegmentColorSolid', () => {
    it('should return first solid color for index 0', () => {
      expect(getSegmentColorSolid(0)).toBe(SEGMENT_COLORS_SOLID[0]);
    });

    it('should return correct solid color for valid indexes', () => {
      for (let i = 0; i < SEGMENT_COLORS_SOLID.length; i++) {
        expect(getSegmentColorSolid(i)).toBe(SEGMENT_COLORS_SOLID[i]);
      }
    });

    it('should wrap around for indexes exceeding array length', () => {
      expect(getSegmentColorSolid(8)).toBe(SEGMENT_COLORS_SOLID[0]);
      expect(getSegmentColorSolid(9)).toBe(SEGMENT_COLORS_SOLID[1]);
    });
  });

  describe('toSolidColor', () => {
    it('should strip alpha from 9-character hex colors', () => {
      expect(toSolidColor('#4299e166')).toBe('#4299e1');
      expect(toSolidColor('#38b2acff')).toBe('#38b2ac');
      expect(toSolidColor('#ed893600')).toBe('#ed8936');
    });

    it('should pass through 7-character hex colors unchanged', () => {
      expect(toSolidColor('#4299e1')).toBe('#4299e1');
      expect(toSolidColor('#ffffff')).toBe('#ffffff');
      expect(toSolidColor('#000000')).toBe('#000000');
    });

    it('should convert rgba to rgb', () => {
      expect(toSolidColor('rgba(66, 153, 225, 0.4)')).toBe(
        'rgb(66, 153, 225)'
      );
      expect(toSolidColor('rgba(0, 0, 0, 1)')).toBe('rgb(0, 0, 0)');
      expect(toSolidColor('rgba(255, 255, 255, 0.5)')).toBe(
        'rgb(255, 255, 255)'
      );
    });

    it('should pass through rgb unchanged', () => {
      expect(toSolidColor('rgb(66, 153, 225)')).toBe('rgb(66, 153, 225)');
    });

    it('should pass through named colors unchanged', () => {
      expect(toSolidColor('red')).toBe('red');
      expect(toSolidColor('blue')).toBe('blue');
      expect(toSolidColor('transparent')).toBe('transparent');
    });

    it('should pass through 3-character hex colors unchanged', () => {
      expect(toSolidColor('#fff')).toBe('#fff');
      expect(toSolidColor('#000')).toBe('#000');
    });

    it('should handle edge cases', () => {
      // Empty string
      expect(toSolidColor('')).toBe('');

      // CSS variables (should pass through)
      expect(toSolidColor('var(--color-blue-500)')).toBe(
        'var(--color-blue-500)'
      );
    });
  });
});

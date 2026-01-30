import { describe, it, expect } from 'vitest';
import { createPayloadValidator, checkType } from '../factory';
import type { FieldValidator } from '../types';

describe('createPayloadValidator', () => {
  describe('basic type validation', () => {
    it('should validate ActionID field correctly', () => {
      const fields: FieldValidator[] = [
        { name: 'id', required: true, type: 'string' },
        { name: 'source_index', required: true, type: 'number' },
      ];

      const validator = createPayloadValidator<unknown>(fields);

      const validPayload = {
        id: 'action-123',
        source_index: 0,
      };

      const result = validator(validPayload);

      expect(result.valid).toBe(true);
      expect(result.data).toEqual(validPayload);
      expect(result.errors).toHaveLength(0);
    });

    it('should validate Role field correctly', () => {
      const fields: FieldValidator[] = [{ name: 'role', required: true, type: 'string' }];

      const validator = createPayloadValidator<unknown>(fields);

      const validPayload = { role: 'Thief' };

      const result = validator(validPayload);

      expect(result.valid).toBe(true);
      expect(result.data).toEqual(validPayload);
    });

    it('should fail when required string field is missing', () => {
      const fields: FieldValidator[] = [
        { name: 'player_id', required: true, type: 'string' },
        { name: 'token', required: true, type: 'string' },
      ];

      const validator = createPayloadValidator<unknown>(fields);

      const invalidPayload = { token: 'valid-token' };

      const result = validator(invalidPayload);

      expect(result.valid).toBe(false);
      expect(result.errors).toHaveLength(1);
      expect(result.errors[0].code).toBe('MISSING_FIELD');
      expect(result.errors[0].path).toBe('player_id');
    });

    it('should fail when string field has wrong type', () => {
      const fields: FieldValidator[] = [{ name: 'player_id', required: true, type: 'string' }];

      const validator = createPayloadValidator<unknown>(fields);

      const invalidPayload = { player_id: 123 };

      const result = validator(invalidPayload);

      expect(result.valid).toBe(false);
      expect(result.errors).toHaveLength(1);
      expect(result.errors[0].code).toBe('TYPE_MISMATCH');
    });
  });

  describe('number validation', () => {
    it('should validate number fields', () => {
      const fields: FieldValidator[] = [
        { name: 'turn_number', required: true, type: 'number' },
        { name: 'active_player_index', required: true, type: 'number' },
      ];

      const validator = createPayloadValidator<unknown>(fields);

      const validPayload = {
        turn_number: 5,
        active_player_index: 0,
      };

      const result = validator(validPayload);

      expect(result.valid).toBe(true);
    });

    it('should fail when number field is not a number', () => {
      const fields: FieldValidator[] = [{ name: 'turn_number', required: true, type: 'number' }];

      const validator = createPayloadValidator<unknown>(fields);

      const invalidPayload = { turn_number: 'five' };

      const result = validator(invalidPayload);

      expect(result.valid).toBe(false);
      expect(result.errors[0].code).toBe('TYPE_MISMATCH');
    });

    it('should reject NaN as invalid number', () => {
      const fields: FieldValidator[] = [{ name: 'value', required: true, type: 'number' }];

      const validator = createPayloadValidator<unknown>(fields);

      const invalidPayload = { value: NaN };

      const result = validator(invalidPayload);

      expect(result.valid).toBe(false);
      expect(result.errors[0].code).toBe('TYPE_MISMATCH');
    });
  });

  describe('boolean validation', () => {
    it('should validate boolean fields', () => {
      const fields: FieldValidator[] = [{ name: 'alive', required: true, type: 'boolean' }];

      const validator = createPayloadValidator<unknown>(fields);

      const validPayload = { alive: true };

      const result = validator(validPayload);

      expect(result.valid).toBe(true);
    });

    it('should fail when boolean field has wrong type', () => {
      const fields: FieldValidator[] = [{ name: 'alive', required: true, type: 'boolean' }];

      const validator = createPayloadValidator<unknown>(fields);

      const invalidPayload = { alive: 'yes' };

      const result = validator(invalidPayload);

      expect(result.valid).toBe(false);
      expect(result.errors[0].code).toBe('TYPE_MISMATCH');
    });
  });

  describe('array validation', () => {
    it('should validate array fields', () => {
      const fields: FieldValidator[] = [
        { name: 'players', required: true, type: 'array' },
      ];

      const validator = createPayloadValidator<unknown>(fields);

      const validPayload = { players: [] };

      const result = validator(validPayload);

      expect(result.valid).toBe(true);
    });

    it('should validate array with item type', () => {
      const fields: FieldValidator[] = [
        { name: 'player_nicks', required: true, type: 'array', itemType: 'string' },
      ];

      const validator = createPayloadValidator<unknown>(fields);

      const validPayload = { player_nicks: ['Alice', 'Bob'] };

      const result = validator(validPayload);

      expect(result.valid).toBe(true);
    });

    it('should fail when array item has wrong type', () => {
      const fields: FieldValidator[] = [
        { name: 'scores', required: true, type: 'array', itemType: 'number' },
      ];

      const validator = createPayloadValidator<unknown>(fields);

      const invalidPayload = { scores: [10, 'twenty', 30] };

      const result = validator(invalidPayload);

      expect(result.valid).toBe(false);
      expect(result.errors[0].code).toBe('INVALID_ARRAY_ITEM');
    });

    it('should fail when field is not an array', () => {
      const fields: FieldValidator[] = [
        { name: 'players', required: true, type: 'array' },
      ];

      const validator = createPayloadValidator<unknown>(fields);

      const invalidPayload = { players: 'not an array' };

      const result = validator(invalidPayload);

      expect(result.valid).toBe(false);
      expect(result.errors[0].code).toBe('TYPE_MISMATCH');
    });
  });

  describe('object validation', () => {
    it('should validate object fields', () => {
      const fields: FieldValidator[] = [
        { name: 'config', required: true, type: 'object' },
      ];

      const validator = createPayloadValidator<unknown>(fields);

      const validPayload = { config: { setting: 'value' } };

      const result = validator(validPayload);

      expect(result.valid).toBe(true);
    });

    it('should fail when object field is null', () => {
      const fields: FieldValidator[] = [
        { name: 'config', required: true, type: 'object' },
      ];

      const validator = createPayloadValidator<unknown>(fields);

      const invalidPayload = { config: null };

      const result = validator(invalidPayload);

      expect(result.valid).toBe(false);
      expect(result.errors[0].code).toBe('TYPE_MISMATCH');
    });

    it('should fail when object field is array', () => {
      const fields: FieldValidator[] = [
        { name: 'config', required: true, type: 'object' },
      ];

      const validator = createPayloadValidator<unknown>(fields);

      const invalidPayload = { config: ['not', 'an', 'object'] };

      const result = validator(invalidPayload);

      expect(result.valid).toBe(false);
      expect(result.errors[0].code).toBe('TYPE_MISMATCH');
    });
  });

  describe('optional fields', () => {
    it('should allow optional fields to be undefined', () => {
      const fields: FieldValidator[] = [
        { name: 'required', required: true, type: 'string' },
        { name: 'optional', required: false, type: 'string' },
      ];

      const validator = createPayloadValidator<unknown>(fields);

      const payload = { required: 'value' };

      const result = validator(payload);

      expect(result.valid).toBe(true);
    });

    it('should allow optional fields to be missing', () => {
      const fields: FieldValidator[] = [
        { name: 'required', required: true, type: 'string' },
        { name: 'optional', required: false, type: 'string' },
      ];

      const validator = createPayloadValidator<unknown>(fields);

      const payload = { required: 'value', optional: undefined };

      const result = validator(payload);

      expect(result.valid).toBe(true);
    });

    it('should validate optional fields when provided', () => {
      const fields: FieldValidator[] = [
        { name: 'optional', required: false, type: 'number' },
      ];

      const validator = createPayloadValidator<unknown>(fields);

      const invalidPayload = { optional: 'not a number' };

      const result = validator(invalidPayload);

      expect(result.valid).toBe(false);
      expect(result.errors[0].code).toBe('TYPE_MISMATCH');
    });
  });

  describe('custom validators', () => {
    it('should use custom validator when provided', () => {
      const fields: FieldValidator[] = [
        {
          name: 'email',
          required: true,
          type: 'string',
          validator: (value: unknown) => typeof value === 'string' && value.includes('@'),
        },
      ];

      const validator = createPayloadValidator<unknown>(fields);

      const validPayload = { email: 'test@example.com' };
      const invalidPayload = { email: 'not-an-email' };

      expect(validator(validPayload).valid).toBe(true);
      expect(validator(invalidPayload).valid).toBe(false);
      expect(validator(invalidPayload).errors[0].code).toBe('CUSTOM_VALIDATION_FAILED');
    });

    it('should use item validator for array items', () => {
      const fields: FieldValidator[] = [
        {
          name: 'emails',
          required: true,
          type: 'array',
          itemType: 'string',
          itemValidator: (item: unknown) => typeof item === 'string' && item.includes('@'),
        },
      ];

      const validator = createPayloadValidator<unknown>(fields);

      const validPayload = { emails: ['a@b.com', 'c@d.com'] };
      const invalidPayload = { emails: ['a@b.com', 'invalid'] };

      expect(validator(validPayload).valid).toBe(true);
      expect(validator(invalidPayload).valid).toBe(false);
      expect(validator(invalidPayload).errors[0].code).toBe('INVALID_ARRAY_ITEM');
    });
  });

  describe('payload must be object', () => {
    it('should fail when payload is not an object', () => {
      const fields: FieldValidator[] = [{ name: 'id', required: true, type: 'string' }];

      const validator = createPayloadValidator<unknown>(fields);

      expect(validator('string').valid).toBe(false);
      expect(validator(123).valid).toBe(false);
      expect(validator(null).valid).toBe(false);
      expect(validator(undefined).valid).toBe(false);
      expect(validator(['array']).valid).toBe(false);
      expect(validator(true).valid).toBe(false);
    });

    it('should return NOT_OBJECT error for non-object payload', () => {
      const fields: FieldValidator[] = [{ name: 'id', required: true, type: 'string' }];

      const validator = createPayloadValidator<unknown>(fields);

      const result = validator('string');

      expect(result.valid).toBe(false);
      expect(result.errors[0].code).toBe('NOT_OBJECT');
    });
  });

  describe('multiple errors', () => {
    it('should collect multiple validation errors', () => {
      const fields: FieldValidator[] = [
        { name: 'id', required: true, type: 'string' },
        { name: 'count', required: true, type: 'number' },
        { name: 'active', required: true, type: 'boolean' },
      ];

      const validator = createPayloadValidator<unknown>(fields);

      const invalidPayload = {
        id: 123, // wrong type
        count: 'many', // wrong type
        // missing active
      };

      const result = validator(invalidPayload);

      expect(result.valid).toBe(false);
      expect(result.errors.length).toBeGreaterThanOrEqual(2);
    });
  });
});

describe('checkType helper', () => {
  it('should check string type', () => {
    const field: FieldValidator = { name: 'test', required: true, type: 'string' };

    expect(checkType('hello', field)).toBe(true);
    expect(checkType(123, field)).toBe(false);
    expect(checkType(null, field)).toBe(false);
  });

  it('should check number type', () => {
    const field: FieldValidator = { name: 'test', required: true, type: 'number' };

    expect(checkType(42, field)).toBe(true);
    expect(checkType(NaN, field)).toBe(false);
    expect(checkType('42', field)).toBe(false);
    expect(checkType(null, field)).toBe(false);
  });

  it('should check boolean type', () => {
    const field: FieldValidator = { name: 'test', required: true, type: 'boolean' };

    expect(checkType(true, field)).toBe(true);
    expect(checkType(false, field)).toBe(true);
    expect(checkType('true', field)).toBe(false);
    expect(checkType(1, field)).toBe(false);
  });

  it('should check array type', () => {
    const field: FieldValidator = { name: 'test', required: true, type: 'array' };

    expect(checkType([], field)).toBe(true);
    expect(checkType([1, 2, 3], field)).toBe(true);
    expect(checkType('array', field)).toBe(false);
    expect(checkType({ length: 0 }, field)).toBe(false);
  });

  it('should check array with item type', () => {
    const field: FieldValidator = {
      name: 'test',
      required: true,
      type: 'array',
      itemType: 'string',
    };

    // checkType only validates that the value is an array
    // Item type validation is handled separately in createPayloadValidator
    expect(checkType(['a', 'b'], field)).toBe(true);
    expect(checkType([1, 2], field)).toBe(true); // checkType doesn't validate items
    expect(checkType(['a', 1], field)).toBe(true); // checkType doesn't validate items
    expect(checkType('not array', field)).toBe(false);
  });

  it('should check object type', () => {
    const field: FieldValidator = { name: 'test', required: true, type: 'object' };

    expect(checkType({}, field)).toBe(true);
    expect(checkType({ key: 'value' }, field)).toBe(true);
    expect(checkType(null, field)).toBe(false);
    expect(checkType([], field)).toBe(false);
    expect(checkType('object', field)).toBe(false);
  });
});

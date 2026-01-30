/**
 * Validator Factory (L3 Validation)
 *
 * Creates payload validators using a declarative field configuration.
 * Supports type checking, custom validation, and array item validation.
 */

import type {
  ValidationResult,
  ValidationError,
  FieldValidator,
} from "./types";

/**
 * Creates a payload validator function from field definitions
 *
 * @param fields - Array of field validator configurations
 * @returns Validator function that checks payload against field definitions
 */
export function createPayloadValidator<T>(
  fields: FieldValidator[]
): (payload: unknown) => ValidationResult<T> {
  return (payload: unknown): ValidationResult<T> => {
    const errors: ValidationError[] = [];

    // Payload must be a non-null object
    if (typeof payload !== "object" || payload === null || Array.isArray(payload)) {
      return {
        valid: false,
        errors: [
          {
            path: "",
            message: "Payload must be an object",
            code: "NOT_OBJECT",
          },
        ],
      };
    }

    const obj = payload as Record<string, unknown>;

    for (const field of fields) {
      const value = obj[field.name];

      // Check required fields
      if (field.required && !(field.name in obj)) {
        errors.push({
          path: field.name,
          message: `Missing required field '${field.name}'`,
          code: "MISSING_FIELD",
        });
        continue;
      }

      // Skip validation for undefined optional fields
      if (value === undefined && !field.required) {
        continue;
      }

      // Type checking
      if (!checkType(value, field)) {
        errors.push({
          path: field.name,
          message: `Field '${field.name}' must be of type ${field.type}`,
          code: "TYPE_MISMATCH",
        });
        continue;
      }

      // Array item validation (separate from type check)
      if (field.type === "array" && Array.isArray(value)) {
        // Check itemType
        if (field.itemType) {
          const hasInvalidItem = value.some(
            (item) => !checkItemType(item, field.itemType!)
          );
          if (hasInvalidItem) {
            errors.push({
              path: field.name,
              message: `Array items in '${field.name}' must be of type ${field.itemType}`,
              code: "INVALID_ARRAY_ITEM",
            });
            continue;
          }
        }

        // Check itemValidator
        if (field.itemValidator) {
          const hasInvalidItem = value.some((item) => !field.itemValidator!(item));
          if (hasInvalidItem) {
            errors.push({
              path: field.name,
              message: `Array items in '${field.name}' failed custom validation`,
              code: "INVALID_ARRAY_ITEM",
            });
            continue;
          }
        }
      }

      // Custom validation
      if (field.validator && !field.validator(value)) {
        errors.push({
          path: field.name,
          message: `Field '${field.name}' failed custom validation`,
          code: "CUSTOM_VALIDATION_FAILED",
        });
      }
    }

    if (errors.length > 0) {
      return { valid: false, errors };
    }

    return { valid: true, data: obj as T, errors: [] };
  };
}

/**
 * Check if a value matches the expected field type
 *
 * @param value - Value to check
 * @param field - Field validator configuration
 * @returns True if value matches the expected type
 */
export function checkType(value: unknown, field: FieldValidator): boolean {
  switch (field.type) {
    case "string":
      return typeof value === "string";

    case "number":
      return typeof value === "number" && !Number.isNaN(value);

    case "boolean":
      return typeof value === "boolean";

    case "array":
      // Just check if it's an array - item validation is handled separately
      return Array.isArray(value);

    case "object":
      return (
        typeof value === "object" && value !== null && !Array.isArray(value)
      );

    default:
      return false;
  }
}

/**
 * Check if an array item matches the expected type
 *
 * @param value - Item value to check
 * @param type - Expected item type
 * @returns True if item matches the expected type
 */
function checkItemType(
  value: unknown,
  type: "string" | "number" | "boolean" | "object"
): boolean {
  switch (type) {
    case "string":
      return typeof value === "string";
    case "number":
      return typeof value === "number" && !Number.isNaN(value);
    case "boolean":
      return typeof value === "boolean";
    case "object":
      return typeof value === "object" && value !== null;
    default:
      return false;
  }
}

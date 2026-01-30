/**
 * Validation Logging and Error Handling
 *
 * Provides structured logging for validation errors across all 4 layers:
 * L1: Structure validation errors
 * L2: Type whitelist errors
 * L3: Payload validation errors
 * L4: Business logic validation errors
 */

import type { ValidationLayer, ValidationError } from "./types";

/**
 * Context for validation error logging
 */
export interface ValidationErrorContext {
  /** Validation layer where error occurred */
  layer: ValidationLayer;
  /** Message type being validated */
  messageType?: string;
  /** Request ID if present */
  requestId?: string;
  /** Error message or summary */
  error: string;
  /** Additional error details */
  details?: ValidationError[];
  /** Raw envelope data (development only) */
  rawEnvelope?: unknown;
}

/**
 * Validation logger interface
 */
export interface ValidationLogger {
  /** Log a validation error */
  logValidationError(context: ValidationErrorContext): void;
  /** Log a validation warning (e.g., unknown message type) */
  logValidationWarning(context: ValidationErrorContext): void;
  /** Log validation metrics (duration) */
  logValidationMetric(messageType: string, duration: number): void;
}

/**
 * Production validation logger
 * - Concise logging
 * - No sensitive data in logs
 * - Suitable for production use
 */
export const productionValidationLogger: ValidationLogger = {
  logValidationError(context: ValidationErrorContext) {
    // In production, we could send to error tracking service (Sentry, etc.)
    console.error("[validation:error]", {
      layer: context.layer,
      messageType: context.messageType,
      requestId: context.requestId,
      error: context.error,
      // Don't log rawEnvelope in production (may contain sensitive data)
      details: context.details,
    });
  },

  logValidationWarning(context: ValidationErrorContext) {
    console.warn("[validation:warning]", {
      layer: context.layer,
      messageType: context.messageType,
      error: context.error,
    });
  },

  logValidationMetric(messageType: string, duration: number) {
    // Send to metrics service if needed
    // console.log(`[validation:metric] ${messageType}: ${duration}ms`);
  },
};

/**
 * Development validation logger
 * - Verbose logging
 * - Includes raw envelope data for debugging
 * - Suitable for development use
 */
export const developmentValidationLogger: ValidationLogger = {
  logValidationError(context: ValidationErrorContext) {
    console.error("[validation:error]", {
      ...context,
      rawEnvelope: context.rawEnvelope, // Include in dev
    });
  },

  logValidationWarning(context: ValidationErrorContext) {
    console.warn("[validation:warning]", context);
  },

  logValidationMetric(messageType: string, duration: number) {
    if (typeof process !== "undefined" && process.env?.NODE_ENV === "development") {
      console.log(`[validation:metric] ${messageType}: ${duration}ms`);
    }
  },
};

/**
 * Active validation logger
 * Uses development logger in development, production logger otherwise
 */
export const validationLogger: ValidationLogger =
  typeof process !== "undefined" && process.env?.NODE_ENV === "production"
    ? productionValidationLogger
    : developmentValidationLogger;

/**
 * Create a validation error context from validation result
 *
 * @param layer - Validation layer
 * @param messageType - Message type being validated
 * @param requestId - Request ID
 * @param errors - Validation errors
 * @returns Validation error context
 */
export function createErrorContext(
  layer: ValidationLayer,
  messageType: string | undefined,
  requestId: string | undefined,
  errors: ValidationError[]
): ValidationErrorContext {
  return {
    layer,
    messageType,
    requestId,
    error: errors.map((e) => `${e.path}: ${e.message}`).join(", "),
    details: errors,
  };
}

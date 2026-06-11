import type { GraphQLError, GraphQLResponse } from "../api/graphql";

export type FieldErrors = Record<string, string>;

export interface ParsedGraphQLErrors {
    /** Whether the response contained any GraphQL errors. */
    hasErrors: boolean;
    /** Per-field validation errors keyed by field name. */
    fieldErrors: FieldErrors;
    /** A top-level (form-wide) error message, empty when only field errors exist. */
    formError: string;
    /** Raw extensions of the first error, for special cases (e.g. lockedUntil). */
    extensions?: GraphQLError["extensions"];
}

/**
 * Normalises a GraphQL response's errors into field errors and a form error.
 * Use this when a page needs to inspect the result (e.g. account-lock handling);
 * otherwise prefer {@link applyGraphQLErrors}.
 */
export function parseGraphQLErrors(
    result: GraphQLResponse,
    fallbackMessage = "Something went wrong. Please try again.",
): ParsedGraphQLErrors {
    const firstError = result.errors?.[0];
    if (!firstError) {
        return { hasErrors: false, fieldErrors: {}, formError: "" };
    }

    const validationErrors = firstError.extensions?.validationErrors;
    if (validationErrors?.length) {
        const fieldErrors: FieldErrors = {};
        for (const item of validationErrors) {
            fieldErrors[item.field] = item.message;
        }
        return { hasErrors: true, fieldErrors, formError: "", extensions: firstError.extensions };
    }

    return {
        hasErrors: true,
        fieldErrors: {},
        formError: firstError.message ?? fallbackMessage,
        extensions: firstError.extensions,
    };
}

export interface ApplyGraphQLErrorsOptions {
    setFieldErrors: (errors: FieldErrors) => void;
    setFormError: (message: string) => void;
    fallbackMessage?: string;
}

/**
 * Parses a GraphQL response and pushes any errors into the supplied React state
 * setters. Returns `true` when errors were present so callers can early-return:
 *
 *   if (applyGraphQLErrors(result, { setFieldErrors, setFormError })) return;
 */
export function applyGraphQLErrors(
    result: GraphQLResponse,
    { setFieldErrors, setFormError, fallbackMessage }: ApplyGraphQLErrorsOptions,
): boolean {
    const parsed = parseGraphQLErrors(result, fallbackMessage);
    if (!parsed.hasErrors) return false;

    if (Object.keys(parsed.fieldErrors).length > 0) {
        setFieldErrors(parsed.fieldErrors);
    }
    if (parsed.formError) {
        setFormError(parsed.formError);
    }
    return true;
}

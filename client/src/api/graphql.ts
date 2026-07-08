import apiClient from "./client"
import { buildHeaders } from "./header";

export interface ValidationError {
	field: string;
	message: string;
}

export interface GraphQLError {
	message: string;
	path?: string[];
	locations?: {
		line: number;
		column: number;
	}[];
	extensions?: {
		validationErrors?: ValidationError[];
		lockedUntil?: string;
	};
}

export interface GraphQLResponse<T = unknown> {
	data?: T | null;
	errors?: GraphQLError[];
}

export const doGraphQL = async <T>(
    query: string,
    accessToken?: string,
): Promise<GraphQLResponse<T>> => {
    try{
        const res = await apiClient.post<GraphQLResponse<T>>(
            "/query",
            {query},
            {headers: buildHeaders(accessToken)}
        );

        const json = res.data
        const errors = Array.isArray(json?.errors) ? json.errors: [];

        return{
            data: json.data ?? null,
            errors,
        }
    }  catch (err: unknown) {
		console.error("GraphQL request failed:", err);
        const error = err as { response?: { data?: { message?: string } } };
		return {
			data: null,
			errors: [
                {message: error.response?.data?.message ?? "Something went wrong"},
			],
		};
	}
}

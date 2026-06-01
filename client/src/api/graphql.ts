import apiClient from "./client"
import { buildHeaders } from "./header";

interface ValidationError {
	field: string;
	message: string;
}

interface GraphQLError {
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

export interface GraphQLResponse<T = any> {
	data?: T | null;
	errors?: GraphQLError[];
}
export const doGraphQL = async <T>(
    query: string,
    accessToken?: string,

): Promise<GraphQLResponse<T>> => {
    try{
        const res = await apiClient.post<any>(
            "/query", 
            {query}, 
            {headers: buildHeaders(accessToken)}
        );

        const json = res.data
        return{
            data: json.data ?? null,
            errors: Array.isArray(json?.errors) ? json.errors: [],
        }
    }  catch (err: any) {
		return {
			data: null,
			errors: [
                {message: err.response?.data?.message ?? "Something went wrong"},
			],
		};
	}
}
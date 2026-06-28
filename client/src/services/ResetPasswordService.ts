import { doGraphQL } from "../api/graphql";

export const resetPassword = async (token: string, newPassword: string) => {
  const query = `
    mutation {
      resetPassword(newPassword: ${JSON.stringify(newPassword)})
    }
  `;

  return doGraphQL<{ resetPassword: boolean }>(query, token);
};

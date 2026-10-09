function trimSlash(url: string): string {
    return url.replace(/\/+$/, "");
}

export const appUrl = trimSlash(import.meta.env.VITE_APP_URL || "https://app.lrsolid.cloud");
export const githubUrl = trimSlash(import.meta.env.VITE_GITHUB_URL || "https://github.com/OmiLAXR-Ecosystem/LRSolid");
export const docsUrl = `${githubUrl}/blob/main/README.md`;
export const xapiUrl = `${appUrl}/xapi/statements`;

export function statementCurl(): string {
    return `curl -u "$KEY_ID:$SECRET" \\
  -H "Content-Type: application/json" \\
  -H "X-Experience-API-Version: 2.0.0" \\
  -X POST "${xapiUrl}" \\
  -d '{
    "actor": { "mbox": "mailto:learner@university.edu" },
    "verb": {
      "id": "http://adlnet.gov/expapi/verbs/completed",
      "display": { "en-US": "completed" }
    },
    "object": { "id": "https://example.edu/activities/intro" }
  }'`;
}

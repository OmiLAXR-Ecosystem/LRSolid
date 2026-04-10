export type RestOrg = {
  id: string;
  slug: string;
  name: string;
  statementCount?: number;
};

export type RestOrgListResponse = {
  items: RestOrg[];
};

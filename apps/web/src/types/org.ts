export type RestOrg = {
  id: string;
  slug: string;
  name: string;
};

export type RestOrgListResponse = {
  items: RestOrg[];
};

import { useEffect, useState } from "react";
import { Card } from "../Card";
import { FieldLabel } from "../ui/FieldLabel";
import { SelectInput } from "../ui/SelectInput";
import { TextInput } from "../ui/TextInput";
import { ApiError } from "../../api/client";
import {
    createAuthProvider,
    deleteAuthProvider,
    getAuthSettings,
    listAuthProviders,
    samlMetadataPath,
    testAuthProvider,
    updateAuthProvider,
    updateAuthSettings,
    type AuthProviderRecord,
    type AuthProviderWrite,
} from "../../api/sso";

const ROLE_OPTIONS = [
    { label: "Viewer", value: "viewer" },
    { label: "Member", value: "member" },
    { label: "Admin", value: "admin" },
    { label: "Owner", value: "owner" },
];

type Draft = {
    id?: string;
    slug: string;
    type: "oidc" | "saml";
    displayName: string;
    enabled: boolean;
    autoProvision: boolean;
    defaultRole: string;
    linkByEmail: boolean;
    issuer: string;
    clientId: string;
    clientSecret: string;
    scopes: string;
    requireEmailVerified: boolean;
    idpMetadataUrl: string;
    idpMetadataXml: string;
    spEntityId: string;
    roleMapping: string;
    generateSpKey: boolean;
};

const emptyDraft = (): Draft => ({
    slug: "",
    type: "oidc",
    displayName: "",
    enabled: true,
    autoProvision: false,
    defaultRole: "viewer",
    linkByEmail: true,
    issuer: "",
    clientId: "",
    clientSecret: "",
    scopes: "openid profile email",
    requireEmailVerified: false,
    idpMetadataUrl: "",
    idpMetadataXml: "",
    spEntityId: "",
    roleMapping: "",
    generateSpKey: false,
});

function draftFrom(p: AuthProviderRecord): Draft {
    const config = p.config ?? {};
    const scopes = Array.isArray(config.scopes) ? config.scopes.join(" ") : "openid profile email";
    return {
        id: p.id,
        slug: p.slug,
        type: p.type,
        displayName: p.displayName,
        enabled: p.enabled,
        autoProvision: p.autoProvision,
        defaultRole: p.defaultRole,
        linkByEmail: p.linkByEmail,
        issuer: String(config.issuer ?? ""),
        clientId: String(config.clientId ?? ""),
        clientSecret: "",
        scopes,
        requireEmailVerified: Boolean(config.requireEmailVerified),
        idpMetadataUrl: String(config.idpMetadataUrl ?? ""),
        idpMetadataXml: String(config.idpMetadataXml ?? ""),
        spEntityId: String(config.spEntityId ?? ""),
        roleMapping: config.roleMapping ? JSON.stringify(config.roleMapping, null, 2) : "",
        generateSpKey: false,
    };
}

function toWrite(draft: Draft): AuthProviderWrite {
    let roleMapping: Record<string, string> | undefined;
    if (draft.roleMapping.trim()) {
        roleMapping = JSON.parse(draft.roleMapping) as Record<string, string>;
    }
    if (draft.type === "oidc") {
        return {
            slug: draft.slug.trim(),
            type: "oidc",
            displayName: draft.displayName.trim(),
            enabled: draft.enabled,
            autoProvision: draft.autoProvision,
            defaultRole: draft.defaultRole,
            linkByEmail: draft.linkByEmail,
            clientSecret: draft.clientSecret.trim() || undefined,
            config: {
                issuer: draft.issuer.trim(),
                clientId: draft.clientId.trim(),
                scopes: draft.scopes.split(/\s+/).filter(Boolean),
                requireEmailVerified: draft.requireEmailVerified,
                roleMapping: roleMapping ?? {},
            },
        };
    }
    return {
        slug: draft.slug.trim(),
        type: "saml",
        displayName: draft.displayName.trim(),
        enabled: draft.enabled,
        autoProvision: draft.autoProvision,
        defaultRole: draft.defaultRole,
        linkByEmail: draft.linkByEmail,
        generateSpKey: draft.generateSpKey,
        config: {
            idpMetadataUrl: draft.idpMetadataUrl.trim(),
            idpMetadataXml: draft.idpMetadataXml.trim(),
            spEntityId: draft.spEntityId.trim(),
            roleMapping: roleMapping ?? {},
        },
    };
}

export function AuthSettingsSection({ org }: { org: string }) {
    const [localLogin, setLocalLogin] = useState(true);
    const [providers, setProviders] = useState<AuthProviderRecord[]>([]);
    const [draft, setDraft] = useState<Draft>(emptyDraft);
    const [error, setError] = useState<string | null>(null);
    const [message, setMessage] = useState<string | null>(null);
    const [saving, setSaving] = useState(false);

    const load = () => {
        setError(null);
        Promise.all([getAuthSettings(org), listAuthProviders(org)])
            .then(([settings, list]) => {
                setLocalLogin(settings.localLoginEnabled);
                setProviders(list.items);
            })
            .catch((err) => setError(err instanceof Error ? err.message : "Failed to load authentication settings"));
    };

    useEffect(() => {
        let cancelled = false;
        setError(null);
        Promise.all([getAuthSettings(org), listAuthProviders(org)])
            .then(([settings, list]) => {
                if (cancelled) return;
                setLocalLogin(settings.localLoginEnabled);
                setProviders(list.items);
            })
            .catch((err) => {
                if (!cancelled) setError(err instanceof Error ? err.message : "Failed to load authentication settings");
            });
        return () => {
            cancelled = true;
        };
    }, [org]);

    const saveLocal = async (enabled: boolean) => {
        setLocalLogin(enabled);
        setError(null);
        try {
            await updateAuthSettings(org, enabled);
            setMessage(enabled ? "Password sign-in enabled." : "Password sign-in disabled.");
        } catch (err) {
            setError(err instanceof ApiError ? err.message : "Failed to save");
        }
    };

    const saveProvider = async () => {
        setSaving(true);
        setError(null);
        setMessage(null);
        try {
            const body = toWrite(draft);
            if (draft.id) {
                await updateAuthProvider(org, draft.id, body);
                setMessage("Provider updated.");
            } else {
                await createAuthProvider(org, body);
                setMessage("Provider created.");
                setDraft(emptyDraft());
            }
            load();
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to save provider");
        } finally {
            setSaving(false);
        }
    };

    const remove = async (id: string) => {
        if (!window.confirm("Delete this sign-in provider?")) return;
        setError(null);
        try {
            await deleteAuthProvider(org, id);
            if (draft.id === id) setDraft(emptyDraft());
            load();
        } catch (err) {
            setError(err instanceof Error ? err.message : "Failed to delete provider");
        }
    };

    const test = async (id: string) => {
        setError(null);
        setMessage(null);
        try {
            const res = await testAuthProvider(org, id);
            setMessage(res.detail || "Connection ok.");
        } catch (err) {
            setError(err instanceof Error ? err.message : "Connection test failed");
        }
    };

    return (
        <Card>
            <div className="space-y-4">
                <div>
                    <div className="text-lg font-semibold text-slate-800">Authentication</div>
                    <div className="text-sm text-slate-500">OpenID Connect and Shibboleth (SAML 2.0) for this organisation.</div>
                </div>
                {error && <div className="text-sm text-red-600">{error}</div>}
                {message && <div className="text-sm text-emerald-700">{message}</div>}
                <label className="flex items-center gap-2 text-sm text-slate-700">
                    <input type="checkbox" checked={localLogin} onChange={(e) => saveLocal(e.target.checked)} />
                    Allow email and password sign-in
                </label>

                <div className="space-y-2">
                    {providers.map((provider) => (
                        <div key={provider.id} className="flex flex-wrap items-center justify-between gap-2 rounded-lg bg-slate-50 px-3 py-2 text-sm">
                            <div>
                                <span className="font-medium text-slate-800">{provider.displayName}</span>
                                <span className="ml-2 text-slate-500">{provider.type} · {provider.slug}</span>
                                {!provider.enabled && <span className="ml-2 text-amber-700">disabled</span>}
                            </div>
                            <div className="flex gap-2">
                                <button type="button" className="text-teal-700" onClick={() => test(provider.id)}>Test</button>
                                {provider.type === "saml" && (
                                    <a className="text-teal-700" href={samlMetadataPath(provider.slug)} target="_blank" rel="noreferrer">Metadata</a>
                                )}
                                <button type="button" className="text-teal-700" onClick={() => setDraft(draftFrom(provider))}>Edit</button>
                                <button type="button" className="text-red-600" onClick={() => remove(provider.id)}>Delete</button>
                            </div>
                        </div>
                    ))}
                    {providers.length === 0 && <div className="text-sm text-slate-500">No SSO providers yet.</div>}
                </div>

                <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                    <div className="space-y-1">
                        <FieldLabel>Type</FieldLabel>
                        <SelectInput
                            value={draft.type}
                            onChange={(value) => setDraft({ ...draft, type: value === "saml" ? "saml" : "oidc" })}
                            options={[{ label: "OpenID Connect", value: "oidc" }, { label: "SAML / Shibboleth", value: "saml" }]}
                        />
                    </div>
                    <div className="space-y-1">
                        <FieldLabel>Display name</FieldLabel>
                        <TextInput value={draft.displayName} onChange={(displayName) => setDraft({ ...draft, displayName })} placeholder="University SSO" />
                    </div>
                    <div className="space-y-1">
                        <FieldLabel>Slug</FieldLabel>
                        <TextInput value={draft.slug} onChange={(slug) => setDraft({ ...draft, slug })} placeholder="shibboleth" />
                    </div>
                    <div className="space-y-1">
                        <FieldLabel>Default role</FieldLabel>
                        <SelectInput value={draft.defaultRole} onChange={(defaultRole) => setDraft({ ...draft, defaultRole })} options={ROLE_OPTIONS} />
                    </div>
                    {draft.type === "oidc" ? (
                        <>
                            <div className="space-y-1 md:col-span-2">
                                <FieldLabel>Issuer</FieldLabel>
                                <TextInput value={draft.issuer} onChange={(issuer) => setDraft({ ...draft, issuer })} placeholder="https://idp.example.edu/realms/main" />
                            </div>
                            <div className="space-y-1">
                                <FieldLabel>Client ID</FieldLabel>
                                <TextInput value={draft.clientId} onChange={(clientId) => setDraft({ ...draft, clientId })} />
                            </div>
                            <div className="space-y-1">
                                <FieldLabel>Client secret</FieldLabel>
                                <TextInput value={draft.clientSecret} onChange={(clientSecret) => setDraft({ ...draft, clientSecret })} type="password" placeholder="Leave blank to keep the current secret" />
                            </div>
                            <div className="space-y-1 md:col-span-2">
                                <FieldLabel>Scopes</FieldLabel>
                                <TextInput value={draft.scopes} onChange={(scopes) => setDraft({ ...draft, scopes })} />
                            </div>
                        </>
                    ) : (
                        <>
                            <div className="space-y-1 md:col-span-2">
                                <FieldLabel>IdP metadata URL</FieldLabel>
                                <TextInput value={draft.idpMetadataUrl} onChange={(idpMetadataUrl) => setDraft({ ...draft, idpMetadataUrl })} placeholder="https://idp.example.edu/idp/shibboleth" />
                            </div>
                            <div className="space-y-1 md:col-span-2">
                                <FieldLabel>IdP metadata XML</FieldLabel>
                                <textarea
                                    value={draft.idpMetadataXml}
                                    onChange={(e) => setDraft({ ...draft, idpMetadataXml: e.target.value })}
                                    rows={4}
                                    className="w-full rounded-lg bg-white px-3 py-2 text-sm text-slate-700 ring-1 ring-slate-200 focus:outline-none focus:ring-2 focus:ring-teal-300"
                                    placeholder="Paste metadata XML if there is no URL"
                                />
                            </div>
                            <div className="space-y-1 md:col-span-2">
                                <FieldLabel>SP entity ID</FieldLabel>
                                <TextInput value={draft.spEntityId} onChange={(spEntityId) => setDraft({ ...draft, spEntityId })} placeholder="Defaults to the metadata URL" />
                            </div>
                        </>
                    )}
                    <div className="space-y-1 md:col-span-2">
                        <FieldLabel>Role mapping JSON</FieldLabel>
                        <textarea
                            value={draft.roleMapping}
                            onChange={(e) => setDraft({ ...draft, roleMapping: e.target.value })}
                            rows={3}
                            className="w-full rounded-lg bg-white px-3 py-2 font-mono text-xs text-slate-700 ring-1 ring-slate-200 focus:outline-none focus:ring-2 focus:ring-teal-300"
                            placeholder={'{\n  "lrs-admins": "admin",\n  "*": "viewer"\n}'}
                        />
                    </div>
                </div>
                <div className="flex flex-wrap items-center gap-4 text-sm text-slate-700">
                    <label className="flex items-center gap-2"><input type="checkbox" checked={draft.enabled} onChange={(e) => setDraft({ ...draft, enabled: e.target.checked })} /> Enabled</label>
                    <label className="flex items-center gap-2"><input type="checkbox" checked={draft.autoProvision} onChange={(e) => setDraft({ ...draft, autoProvision: e.target.checked })} /> Auto-provision</label>
                    <label className="flex items-center gap-2"><input type="checkbox" checked={draft.linkByEmail} onChange={(e) => setDraft({ ...draft, linkByEmail: e.target.checked })} /> Link by verified email</label>
                    {draft.type === "oidc" && (
                        <label className="flex items-center gap-2"><input type="checkbox" checked={draft.requireEmailVerified} onChange={(e) => setDraft({ ...draft, requireEmailVerified: e.target.checked })} /> Require email_verified</label>
                    )}
                    {draft.type === "saml" && (
                        <label className="flex items-center gap-2"><input type="checkbox" checked={draft.generateSpKey} onChange={(e) => setDraft({ ...draft, generateSpKey: e.target.checked })} /> Generate new SP key</label>
                    )}
                </div>
                <div className="flex justify-end gap-2">
                    {draft.id && (
                        <button type="button" className="rounded-lg px-4 py-2 text-sm text-slate-600" onClick={() => setDraft(emptyDraft())}>Cancel</button>
                    )}
                    <button
                        type="button"
                        onClick={saveProvider}
                        disabled={saving}
                        className="rounded-lg bg-teal-600 px-4 py-2 text-sm font-semibold text-white hover:bg-teal-700 disabled:opacity-60"
                    >
                        {saving ? "Saving..." : draft.id ? "Update provider" : "Add provider"}
                    </button>
                </div>
            </div>
        </Card>
    );
}

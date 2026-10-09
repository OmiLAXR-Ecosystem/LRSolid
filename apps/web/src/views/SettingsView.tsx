import { useEffect, useState } from "react";
import { Card } from "../components/Card";
import { FieldLabel } from "../components/ui/FieldLabel";
import { TextInput } from "../components/ui/TextInput";
import { SelectInput } from "../components/ui/SelectInput";
import { useAuth } from "../contexts/AuthContext";
import { useOrg } from "../contexts/OrgContext";
import { fetchExportSettings, updateExportSettings } from "../api/export";
import { adminAssignUserToOrg, adminCreateOrg, adminCreateUser } from "../api/admin";
import { ApiError } from "../api/client";
import { InviteUserDialog } from "../components/users/InviteUserDialog";
import { AuthSettingsSection } from "../components/settings/AuthSettingsSection";
import { DataStoresSection } from "../components/settings/DataStoresSection";
import { LimitsEditor } from "../components/limits/LimitsPanel";
import type { ExportSettings } from "../types/export";

function toGB(n: number): string {
  return String(Math.round((n / (1024 * 1024 * 1024)) * 100) / 100);
}

function fromGB(s: string): number | null {
  const v = Number(s);
  if (!Number.isFinite(v) || v < 0) return null;
  return Math.round(v * 1024 * 1024 * 1024);
}

function slugifyOrgName(name: string): string {
  const base = name
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
  return base || "org";
}

function randomShortID(): string {
  return Math.random().toString(16).slice(2, 8).padEnd(6, "0");
}

const ROLE_OPTIONS = [
  { label: "Viewer", value: "viewer" },
  { label: "Member", value: "member" },
  { label: "Admin", value: "admin" },
  { label: "Owner", value: "owner" },
];

export function SettingsView() {
  const { state } = useAuth();
  const { currentOrg, orgs } = useOrg();
  const canManageAdmin = state.status === "authenticated" && (state.user.role === "admin" || state.user.role === "owner");

  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [remoteThresholdGB, setRemoteThresholdGB] = useState("2");
  const [localMaxGB, setLocalMaxGB] = useState("2");
  const [defaultTargetType, setDefaultTargetType] = useState<ExportSettings["defaultTargetType"]>("s3");
  const [s3UploadBaseUrl, setS3UploadBaseUrl] = useState("");
  const [s3AccessKey, setS3AccessKey] = useState("");
  const [s3SecretKey, setS3SecretKey] = useState("");
  const [s3Region, setS3Region] = useState("us-east-1");
  const [s3AuthToken, setS3AuthToken] = useState("");
  const [s3PathPrefix, setS3PathPrefix] = useState("exports");

  const [inviteOpen, setInviteOpen] = useState(false);
  const [inviteOrg, setInviteOrg] = useState("");

  const [newOrgName, setNewOrgName] = useState("");
  const [newOrgSlug, setNewOrgSlug] = useState("");
  const [orgSlugRandomSuffix, setOrgSlugRandomSuffix] = useState(() => randomShortID());
  const [createOrgLoading, setCreateOrgLoading] = useState(false);
  const [createOrgError, setCreateOrgError] = useState<string | null>(null);
  const [createOrgMessage, setCreateOrgMessage] = useState<string | null>(null);

  const [createUserEmail, setCreateUserEmail] = useState("");
  const [createUserName, setCreateUserName] = useState("");
  const [createUserPassword, setCreateUserPassword] = useState("");
  const [createUserRole, setCreateUserRole] = useState("member");
  const [createUserOrg, setCreateUserOrg] = useState("");
  const [createUserLoading, setCreateUserLoading] = useState(false);
  const [createUserError, setCreateUserError] = useState<string | null>(null);
  const [createUserMessage, setCreateUserMessage] = useState<string | null>(null);

  const [assignEmail, setAssignEmail] = useState("");
  const [assignRole, setAssignRole] = useState("member");
  const [assignOrg, setAssignOrg] = useState("");
  const [assignLoading, setAssignLoading] = useState(false);
  const [assignError, setAssignError] = useState<string | null>(null);
  const [assignMessage, setAssignMessage] = useState<string | null>(null);

  const orgOptions = orgs.map((org) => ({ label: `${org.name} (${org.slug})`, value: org.slug }));
  const currentOrgRecord = orgs.find((org) => org.slug === currentOrg);
  const generatedOrgSlug = `${slugifyOrgName(newOrgName)}-${orgSlugRandomSuffix}`;

  useEffect(() => {
    if (!currentOrg) return;
    setLoading(true);
    setError(null);
    fetchExportSettings(currentOrg)
      .then((s) => {
        setRemoteThresholdGB(toGB(s.requireRemoteTargetAboveBytes));
        setLocalMaxGB(toGB(s.localMaxBytes));
        setDefaultTargetType(s.defaultTargetType === "local" ? "local" : "s3");
        setS3UploadBaseUrl(s.s3UploadBaseUrl ?? "");
        setS3AccessKey(s.s3AccessKey ?? "");
        setS3SecretKey(s.s3SecretKey ?? "");
        setS3Region(s.s3Region ?? "us-east-1");
        setS3AuthToken(s.s3AuthToken ?? "");
        setS3PathPrefix(s.s3PathPrefix ?? "exports");
      })
      .catch((err) => setError(err instanceof Error ? err.message : "Failed to load settings"))
      .finally(() => setLoading(false));
  }, [currentOrg]);

  useEffect(() => {
    const fallbackOrg = currentOrg || orgs[0]?.slug || "";
    if (!fallbackOrg) return;
    if (!inviteOrg || !orgs.some((org) => org.slug === inviteOrg)) setInviteOrg(fallbackOrg);
    if (!createUserOrg || !orgs.some((org) => org.slug === createUserOrg)) setCreateUserOrg(fallbackOrg);
    if (!assignOrg || !orgs.some((org) => org.slug === assignOrg)) setAssignOrg(fallbackOrg);
  }, [assignOrg, createUserOrg, currentOrg, inviteOrg, orgs]);

  const save = async () => {
    if (!currentOrg) return;
    setSaving(true);
    setError(null);
    setMessage(null);
    const requireRemoteTargetAboveBytes = fromGB(remoteThresholdGB);
    const localMaxBytes = fromGB(localMaxGB);
    if (requireRemoteTargetAboveBytes === null || localMaxBytes === null) {
      setSaving(false);
      setError("Please enter valid non-negative GB values.");
      return;
    }
    try {
      await updateExportSettings(currentOrg, {
        requireRemoteTargetAboveBytes,
        localMaxBytes,
        defaultTargetType,
        s3UploadBaseUrl,
        s3AccessKey,
        s3SecretKey,
        s3Region,
        s3AuthToken,
        s3PathPrefix,
      });
      setMessage("Export settings saved.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save settings");
    } finally {
      setSaving(false);
    }
  };

  const createOrg = async () => {
    setCreateOrgError(null);
    setCreateOrgMessage(null);
    if (!newOrgName.trim()) {
      setCreateOrgError("Name ist erforderlich.");
      return;
    }

    const requestedSlug = newOrgSlug.trim() || generatedOrgSlug;
    setCreateOrgLoading(true);
    try {
      const created = await adminCreateOrg({ name: newOrgName.trim(), slug: requestedSlug });
      setCreateOrgMessage(`Organisation ${created.name} (${created.slug}) wurde erstellt.`);
      setNewOrgName("");
      setNewOrgSlug("");
      setOrgSlugRandomSuffix(randomShortID());
    } catch (err) {
      setCreateOrgError(err instanceof ApiError ? err.message : "Organisation konnte nicht erstellt werden.");
    } finally {
      setCreateOrgLoading(false);
    }
  };

  const createUser = async () => {
    setCreateUserError(null);
    setCreateUserMessage(null);
    if (!createUserEmail.trim() || !createUserPassword.trim() || !createUserOrg) {
      setCreateUserError("E-Mail, Passwort und Organisation sind erforderlich.");
      return;
    }
    setCreateUserLoading(true);
    try {
      const created = await adminCreateUser({
        email: createUserEmail.trim(),
        displayName: createUserName.trim() || undefined,
        password: createUserPassword,
        role: createUserRole,
        orgSlug: createUserOrg,
      });
      setCreateUserMessage(`User ${created.email} wurde erstellt und ${createUserOrg} zugewiesen.`);
      setCreateUserEmail("");
      setCreateUserName("");
      setCreateUserPassword("");
      setCreateUserRole("member");
    } catch (err) {
      setCreateUserError(err instanceof ApiError ? err.message : "User konnte nicht erstellt werden.");
    } finally {
      setCreateUserLoading(false);
    }
  };

  const assignUser = async () => {
    setAssignError(null);
    setAssignMessage(null);
    if (!assignEmail.trim() || !assignOrg) {
      setAssignError("E-Mail und Organisation sind erforderlich.");
      return;
    }
    setAssignLoading(true);
    try {
      const assigned = await adminAssignUserToOrg({
        email: assignEmail.trim(),
        orgSlug: assignOrg,
        role: assignRole,
      });
      setAssignMessage(`User ${assigned.email} wurde ${assigned.orgSlug} als ${assigned.role} zugewiesen.`);
      setAssignEmail("");
      setAssignRole("member");
    } catch (err) {
      setAssignError(err instanceof ApiError ? err.message : "Zuweisung fehlgeschlagen.");
    } finally {
      setAssignLoading(false);
    }
  };

  return (
    <div className="space-y-6">
      <Card>
        <div className="space-y-4">
          <div>
            <div className="text-lg font-semibold text-slate-800">Export Settings</div>
            <div className="text-sm text-slate-500">Threshold policy for large statement exports.</div>
          </div>
          {loading && <div className="text-sm text-slate-500">Loading...</div>}
          {error && <div className="text-sm text-red-600">{error}</div>}
          {message && <div className="text-sm text-emerald-700">{message}</div>}
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <div className="space-y-1">
              <FieldLabel>Require Remote Target Above (GB)</FieldLabel>
              <TextInput value={remoteThresholdGB} onChange={setRemoteThresholdGB} type="number" />
            </div>
            <div className="space-y-1">
              <FieldLabel>Local Export Max (GB)</FieldLabel>
              <TextInput value={localMaxGB} onChange={setLocalMaxGB} type="number" />
            </div>
            <div className="space-y-1">
              <FieldLabel>Default Export Target</FieldLabel>
              <select
                value={defaultTargetType}
                onChange={(e) => setDefaultTargetType(e.target.value as ExportSettings["defaultTargetType"])}
                className="w-full rounded-lg bg-white px-3 py-2 text-sm text-slate-700 ring-1 ring-slate-200 focus:ring-2 focus:ring-teal-300 focus:outline-none"
              >
                <option value="local">Local</option>
                <option value="s3">S3</option>
                <option value="gdrive" disabled>
                  Google Drive (not implemented)
                </option>
                <option value="onedrive" disabled>
                  OneDrive (not implemented)
                </option>
                <option value="icloud" disabled>
                  iCloud (not implemented)
                </option>
              </select>
            </div>
            <div className="space-y-1 md:col-span-2">
              <FieldLabel>S3 Upload Base URL</FieldLabel>
              <TextInput value={s3UploadBaseUrl} onChange={setS3UploadBaseUrl} placeholder="https://s3.example.com/bucket" />
            </div>
            <div className="space-y-1">
              <FieldLabel>S3 Access Key</FieldLabel>
              <TextInput value={s3AccessKey} onChange={setS3AccessKey} placeholder="AKIA..." />
            </div>
            <div className="space-y-1">
              <FieldLabel>S3 Secret Key</FieldLabel>
              <TextInput value={s3SecretKey} onChange={setS3SecretKey} placeholder="Secret access key" />
            </div>
            <div className="space-y-1">
              <FieldLabel>S3 Region</FieldLabel>
              <TextInput value={s3Region} onChange={setS3Region} placeholder="us-east-1" />
            </div>
            <div className="space-y-1">
              <FieldLabel>S3 Auth Token (optional, non-AWS endpoints)</FieldLabel>
              <TextInput value={s3AuthToken} onChange={setS3AuthToken} placeholder="Bearer token (without prefix)" />
            </div>
            <div className="space-y-1 md:col-span-2">
              <FieldLabel>S3 Path Prefix</FieldLabel>
              <TextInput value={s3PathPrefix} onChange={setS3PathPrefix} placeholder="exports" />
            </div>
          </div>
          <div className="flex justify-end">
            <button
              onClick={save}
              disabled={saving || loading}
              className="rounded-lg bg-teal-600 px-4 py-2 text-sm font-semibold text-white hover:bg-teal-700 disabled:opacity-60"
            >
              {saving ? "Saving..." : "Save"}
            </button>
          </div>
        </div>
      </Card>

      {currentOrgRecord && (
        <Card>
          <div className="space-y-4">
            <div>
              <div className="text-lg font-semibold text-slate-800">Usage limits</div>
              <div className="text-sm text-slate-500">Caps for the whole organization. Project, LRS, and client limits apply as well.</div>
            </div>
            <LimitsEditor
              org={currentOrg}
              subjectType="organisation"
              subjectId={currentOrgRecord.id}
              readOnly={!canManageAdmin}
            />
          </div>
        </Card>
      )}

      {canManageAdmin && currentOrg && <DataStoresSection org={currentOrg} />}
      {canManageAdmin && currentOrg && <AuthSettingsSection org={currentOrg} />}

      {canManageAdmin && (
        <>
          <Card>
            <div className="space-y-4">
              <div>
                <div className="text-lg font-semibold text-slate-800">Admin Panel</div>
                <div className="text-sm text-slate-500">Organisationen und Benutzer zentral verwalten.</div>
              </div>
              <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                <div className="space-y-1">
                  <FieldLabel>Einladung Organisation</FieldLabel>
                  <SelectInput value={inviteOrg} onChange={setInviteOrg} options={orgOptions} />
                </div>
                <div className="flex items-end">
                  <button
                    onClick={() => setInviteOpen(true)}
                    disabled={!inviteOrg}
                    className="h-10 rounded-lg bg-teal-600 px-4 text-sm font-semibold text-white hover:bg-teal-700 disabled:opacity-60"
                  >
                    User einladen
                  </button>
                </div>
              </div>
            </div>
          </Card>

          <Card>
            <div className="space-y-4">
              <div className="text-base font-semibold text-slate-800">Organisation anlegen</div>
              {createOrgError && <div className="text-sm text-red-600">{createOrgError}</div>}
              {createOrgMessage && <div className="text-sm text-emerald-700">{createOrgMessage}</div>}
              <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                <div className="space-y-1">
                  <FieldLabel>Name</FieldLabel>
                  <TextInput value={newOrgName} onChange={setNewOrgName} placeholder="Acme GmbH" />
                </div>
                <div className="space-y-1">
                  <FieldLabel>Slug</FieldLabel>
                  <TextInput value={newOrgSlug} onChange={setNewOrgSlug} placeholder={generatedOrgSlug} />
                </div>
              </div>
              <div className="flex justify-end">
                <button
                  onClick={createOrg}
                  disabled={createOrgLoading}
                  className="rounded-lg bg-teal-600 px-4 py-2 text-sm font-semibold text-white hover:bg-teal-700 disabled:opacity-60"
                >
                  {createOrgLoading ? "Erstelle..." : "Organisation erstellen"}
                </button>
              </div>
            </div>
          </Card>

          <Card>
            <div className="space-y-4">
              <div className="text-base font-semibold text-slate-800">User anlegen und zuweisen</div>
              {createUserError && <div className="text-sm text-red-600">{createUserError}</div>}
              {createUserMessage && <div className="text-sm text-emerald-700">{createUserMessage}</div>}
              <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                <div className="space-y-1">
                  <FieldLabel>E-Mail</FieldLabel>
                  <TextInput value={createUserEmail} onChange={setCreateUserEmail} placeholder="user@example.com" type="email" />
                </div>
                <div className="space-y-1">
                  <FieldLabel>Name (optional)</FieldLabel>
                  <TextInput value={createUserName} onChange={setCreateUserName} placeholder="Max Mustermann" />
                </div>
                <div className="space-y-1">
                  <FieldLabel>Passwort</FieldLabel>
                  <TextInput value={createUserPassword} onChange={setCreateUserPassword} type="password" />
                </div>
                <div className="space-y-1">
                  <FieldLabel>Rolle</FieldLabel>
                  <SelectInput value={createUserRole} onChange={setCreateUserRole} options={ROLE_OPTIONS} />
                </div>
                <div className="space-y-1">
                  <FieldLabel>Organisation</FieldLabel>
                  <SelectInput value={createUserOrg} onChange={setCreateUserOrg} options={orgOptions} />
                </div>
              </div>
              <div className="flex justify-end">
                <button
                  onClick={createUser}
                  disabled={createUserLoading}
                  className="rounded-lg bg-teal-600 px-4 py-2 text-sm font-semibold text-white hover:bg-teal-700 disabled:opacity-60"
                >
                  {createUserLoading ? "Erstelle..." : "User erstellen"}
                </button>
              </div>
            </div>
          </Card>

          <Card>
            <div className="space-y-4">
              <div className="text-base font-semibold text-slate-800">Bestehenden User zuweisen</div>
              {assignError && <div className="text-sm text-red-600">{assignError}</div>}
              {assignMessage && <div className="text-sm text-emerald-700">{assignMessage}</div>}
              <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
                <div className="space-y-1">
                  <FieldLabel>User E-Mail</FieldLabel>
                  <TextInput value={assignEmail} onChange={setAssignEmail} placeholder="user@example.com" type="email" />
                </div>
                <div className="space-y-1">
                  <FieldLabel>Organisation</FieldLabel>
                  <SelectInput value={assignOrg} onChange={setAssignOrg} options={orgOptions} />
                </div>
                <div className="space-y-1">
                  <FieldLabel>Rolle</FieldLabel>
                  <SelectInput value={assignRole} onChange={setAssignRole} options={ROLE_OPTIONS} />
                </div>
              </div>
              <div className="flex justify-end">
                <button
                  onClick={assignUser}
                  disabled={assignLoading}
                  className="rounded-lg bg-teal-600 px-4 py-2 text-sm font-semibold text-white hover:bg-teal-700 disabled:opacity-60"
                >
                  {assignLoading ? "Speichere..." : "Zuweisen"}
                </button>
              </div>
            </div>
          </Card>
        </>
      )}

      {inviteOrg && (
        <InviteUserDialog
          open={inviteOpen}
          org={inviteOrg}
          onClose={() => setInviteOpen(false)}
          onSuccess={() => {
            setInviteOpen(false);
            setMessage("Einladung wurde erstellt.");
          }}
        />
      )}
    </div>
  );
}

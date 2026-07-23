import { useEffect, useState } from "react";
import { Card } from "../components/Card";
import { SelectInput } from "../components/ui/SelectInput";
import { TextInput } from "../components/ui/TextInput";
import { useAuth } from "../contexts/AuthContext";

type ProfilePreferences = {
    displayName: string;
    language: "de" | "en";
    timezone: string;
    emailNotifications: boolean;
};

const PROFILE_PREFERENCES_KEY = "lrsolid.profile.preferences.v1";

function getDefaultTimezone(): string {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC";
}

function loadProfilePreferences(): ProfilePreferences {
    const fallback: ProfilePreferences = {
        displayName: "",
        language: "de",
        timezone: getDefaultTimezone(),
        emailNotifications: true,
    };

    try {
        const raw = window.localStorage.getItem(PROFILE_PREFERENCES_KEY);
        if (!raw) return fallback;
        const parsed = JSON.parse(raw) as Partial<ProfilePreferences>;
        return {
            displayName: typeof parsed.displayName === "string" ? parsed.displayName : fallback.displayName,
            language: parsed.language === "en" ? "en" : "de",
            timezone: typeof parsed.timezone === "string" && parsed.timezone.trim() ? parsed.timezone : fallback.timezone,
            emailNotifications: typeof parsed.emailNotifications === "boolean" ? parsed.emailNotifications : fallback.emailNotifications,
        };
    } catch {
        return fallback;
    }
}

export function ProfileView() {
    const { state } = useAuth();
    const [prefs, setPrefs] = useState<ProfilePreferences>(loadProfilePreferences);
    const [message, setMessage] = useState<string | null>(null);

    useEffect(() => {
        setMessage(null);
    }, [prefs]);

    const savePreferences = () => {
        window.localStorage.setItem(PROFILE_PREFERENCES_KEY, JSON.stringify(prefs));
        setMessage("Profile settings gespeichert.");
    };

    if (state.status !== "authenticated") return null;

    return (
        <div className="space-y-6">
            <Card title="Profile">
                <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                    <div className="space-y-1">
                        <div className="text-xs font-semibold uppercase tracking-wide text-slate-500">E-Mail</div>
                        <div className="rounded-lg bg-slate-50 px-3 py-2 text-sm text-slate-700 ring-1 ring-slate-200">
                            {state.user.email}
                        </div>
                    </div>
                    <div className="space-y-1">
                        <div className="text-xs font-semibold uppercase tracking-wide text-slate-500">Rolle</div>
                        <div className="rounded-lg bg-slate-50 px-3 py-2 text-sm capitalize text-slate-700 ring-1 ring-slate-200">
                            {state.user.role}
                        </div>
                    </div>
                </div>
            </Card>

            <Card title="Persoenliche Einstellungen">
                <div className="space-y-4">
                    <div className="space-y-1">
                        <div className="text-xs font-semibold uppercase tracking-wide text-slate-500">Anzeigename</div>
                        <TextInput value={prefs.displayName} onChange={(displayName) => setPrefs((prev) => ({ ...prev, displayName }))} />
                    </div>
                    <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                        <div className="space-y-1">
                            <div className="text-xs font-semibold uppercase tracking-wide text-slate-500">Sprache</div>
                            <SelectInput
                                value={prefs.language}
                                onChange={(language) => setPrefs((prev) => ({ ...prev, language: language === "en" ? "en" : "de" }))}
                                options={[
                                    { label: "Deutsch", value: "de" },
                                    { label: "English", value: "en" },
                                ]}
                            />
                        </div>
                        <div className="space-y-1">
                            <div className="text-xs font-semibold uppercase tracking-wide text-slate-500">Zeitzone</div>
                            <TextInput value={prefs.timezone} onChange={(timezone) => setPrefs((prev) => ({ ...prev, timezone }))} />
                        </div>
                    </div>
                    <label className="flex items-center gap-2 text-sm text-slate-700">
                        <input
                            type="checkbox"
                            checked={prefs.emailNotifications}
                            onChange={(e) => setPrefs((prev) => ({ ...prev, emailNotifications: e.target.checked }))}
                            className="h-4 w-4 rounded border-slate-300 text-teal-600 focus:ring-teal-500"
                        />
                        E-Mail-Benachrichtigungen aktivieren
                    </label>
                    <div className="flex items-center gap-3">
                        <button
                            type="button"
                            onClick={savePreferences}
                            className="rounded-lg bg-teal-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-teal-700"
                        >
                            Speichern
                        </button>
                        {message && <div className="text-sm text-emerald-700">{message}</div>}
                    </div>
                </div>
            </Card>
        </div>
    );
}

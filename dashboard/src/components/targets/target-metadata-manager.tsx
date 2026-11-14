
"use client";

import { useCallback, useEffect, useState } from "react";
import type { FormEvent } from "react";
import {
  Layers3,
  Plus,
  Save,
  Tag,
} from "lucide-react";

import type {
  TargetGroup,
  TargetMetadata,
  TargetTag,
} from "@/lib/api/types";

interface TargetMetadataManagerProps {
  probeId: string;
  target: string;
}

interface TargetMetadataData {
  groups: TargetGroup[];
  tags: TargetTag[];
  metadata: TargetMetadata;
}

async function readError(response: Response): Promise<string> {
  const body = await response.json().catch(() => null);
  return body?.error ?? "Something went wrong.";
}

// Fungsi ini hanya mengambil data, tidak mengubah React state.
async function fetchTargetMetadata(
  probeId: string,
  target: string,
  signal?: AbortSignal,
): Promise<TargetMetadataData> {
  const [groupsResponse, tagsResponse, metadataResponse] =
    await Promise.all([
      fetch("/api/target-groups", {
        cache: "no-store",
        signal,
      }),
      fetch("/api/target-tags", {
        cache: "no-store",
        signal,
      }),
      fetch(
        `/api/target-metadata?${new URLSearchParams({
          probe_id: probeId,
          target,
        })}`,
        {
          cache: "no-store",
          signal,
        },
      ),
    ]);

  if (!groupsResponse.ok) {
    throw new Error(await readError(groupsResponse));
  }

  if (!tagsResponse.ok) {
    throw new Error(await readError(tagsResponse));
  }

  if (!metadataResponse.ok) {
    throw new Error(await readError(metadataResponse));
  }

  const groupsResult: { data: TargetGroup[] } =
    await groupsResponse.json();

  const tagsResult: { data: TargetTag[] } =
    await tagsResponse.json();

  const metadata: TargetMetadata =
    await metadataResponse.json();

  return {
    groups: groupsResult.data,
    tags: tagsResult.data,
    metadata,
  };
}

export default function TargetMetadataManager({
  probeId,
  target,
}: TargetMetadataManagerProps) {
  const [groups, setGroups] = useState<TargetGroup[]>([]);
  const [tags, setTags] = useState<TargetTag[]>([]);
  const [selectedGroup, setSelectedGroup] = useState("");
  const [selectedTags, setSelectedTags] = useState<number[]>([]);

  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [creatingGroup, setCreatingGroup] = useState(false);
  const [creatingTag, setCreatingTag] = useState(false);

  const [groupName, setGroupName] = useState("");
  const [groupDescription, setGroupDescription] = useState("");
  const [tagName, setTagName] = useState("");

  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  // Initial data loading.
  // State updates happen in Promise callbacks, not synchronously in the effect.
  useEffect(() => {
    const controller = new AbortController();
    let active = true;

    fetchTargetMetadata(probeId, target, controller.signal)
      .then((result) => {
        if (!active) return;

        setGroups(result.groups);
        setTags(result.tags);

        setSelectedGroup(
          result.metadata.group
            ? String(result.metadata.group.id)
            : "",
        );

        setSelectedTags(
          result.metadata.tags.map((tag) => tag.id),
        );

        setError("");
      })
      .catch((err: unknown) => {
        if (!active) return;

        setError(
          err instanceof Error
            ? err.message
            : "Failed to load metadata.",
        );
      })
      .finally(() => {
        if (active) {
          setLoading(false);
        }
      });

    return () => {
      active = false;
      controller.abort();
    };
  }, [probeId, target]);

  // Reusable function for applying fetched data.
  const applyMetadata = useCallback(
    (result: TargetMetadataData) => {
      setGroups(result.groups);
      setTags(result.tags);

      setSelectedGroup(
        result.metadata.group
          ? String(result.metadata.group.id)
          : "",
      );

      setSelectedTags(
        result.metadata.tags.map((tag) => tag.id),
      );
    },
    [],
  );

  async function handleReload() {
    setLoading(true);
    setError("");
    setSuccess("");

    try {
      const result = await fetchTargetMetadata(probeId, target);

      applyMetadata(result);
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Failed to load metadata.",
      );
    } finally {
      setLoading(false);
    }
  }

  function toggleTag(tagId: number) {
    setSelectedTags((current) =>
      current.includes(tagId)
        ? current.filter((id) => id !== tagId)
        : [...current, tagId],
    );
  }

  async function handleCreateGroup(
    event: FormEvent<HTMLFormElement>,
  ) {
    event.preventDefault();

    setCreatingGroup(true);
    setError("");
    setSuccess("");

    try {
      const response = await fetch("/api/target-groups", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          name: groupName,
          description: groupDescription,
        }),
      });

      if (!response.ok) {
        throw new Error(await readError(response));
      }

      const created: TargetGroup = await response.json();

      setGroups((current) =>
        [...current, created].sort((a, b) =>
          a.name.localeCompare(b.name),
        ),
      );

      setSelectedGroup(String(created.id));
      setGroupName("");
      setGroupDescription("");

      setSuccess(
        "Group created. Save changes to assign it to this target.",
      );
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Failed to create group.",
      );
    } finally {
      setCreatingGroup(false);
    }
  }

  async function handleCreateTag(
    event: FormEvent<HTMLFormElement>,
  ) {
    event.preventDefault();

    setCreatingTag(true);
    setError("");
    setSuccess("");

    try {
      const response = await fetch("/api/target-tags", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          name: tagName,
        }),
      });

      if (!response.ok) {
        throw new Error(await readError(response));
      }

      const created: TargetTag = await response.json();

      setTags((current) =>
        [...current, created].sort((a, b) =>
          a.name.localeCompare(b.name),
        ),
      );

      setSelectedTags((current) =>
        current.includes(created.id)
          ? current
          : [...current, created.id],
      );

      setTagName("");

      setSuccess(
        "Tag created. Save changes to assign it to this target.",
      );
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Failed to create tag.",
      );
    } finally {
      setCreatingTag(false);
    }
  }

  async function handleSave() {
    setSaving(true);
    setError("");
    setSuccess("");

    try {
      const response = await fetch("/api/target-metadata", {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          probe_id: probeId,
          target,
          group_id: selectedGroup
            ? Number(selectedGroup)
            : null,
          tags: selectedTags,
        }),
      });

      if (!response.ok) {
        throw new Error(await readError(response));
      }

      const metadata: TargetMetadata =
        await response.json();

      setSelectedGroup(
        metadata.group
          ? String(metadata.group.id)
          : "",
      );

      setSelectedTags(
        metadata.tags.map((tag) => tag.id),
      );

      setSuccess("Target metadata saved successfully.");
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Failed to save metadata.",
      );
    } finally {
      setSaving(false);
    }
  }

  const isBusy =
    saving || creatingGroup || creatingTag;

  return (
    <section className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
      <div className="mb-5">
        <h2 className="font-semibold text-slate-900">
          Target Groups & Tags
        </h2>

        <p className="mt-1 text-xs text-slate-500">
          Organize this target with a group and descriptive tags.
        </p>
      </div>

      {loading ? (
        <div className="py-8 text-center text-sm text-slate-500">
          Loading target metadata...
        </div>
      ) : (
        <div className="space-y-6">
          {/* Group Selection */}
          <div>
            <label
              htmlFor="target-group"
              className="mb-2 flex items-center gap-2 text-sm font-medium text-slate-700"
            >
              <Layers3 size={16} />
              Group
            </label>

            <select
              id="target-group"
              value={selectedGroup}
              onChange={(event) =>
                setSelectedGroup(event.target.value)
              }
              disabled={isBusy}
              className="h-10 w-full rounded-lg border border-slate-200 bg-white px-3 text-sm text-slate-700 outline-none focus:border-blue-500 disabled:opacity-50"
            >
              <option value="">No group</option>

              {groups.map((group) => (
                <option key={group.id} value={group.id}>
                  {group.name}
                </option>
              ))}
            </select>
          </div>

          {/* Tag Selection */}
          <div>
            <label className="mb-3 flex items-center gap-2 text-sm font-medium text-slate-700">
              <Tag size={16} />
              Tags
            </label>

            {tags.length === 0 ? (
              <p className="text-sm text-slate-400">
                No tags available. Create your first tag below.
              </p>
            ) : (
              <div className="flex flex-wrap gap-2">
                {tags.map((tag) => {
                  const checked = selectedTags.includes(tag.id);

                  return (
                    <label
                      key={tag.id}
                      className={`inline-flex cursor-pointer items-center gap-2 rounded-lg border px-3 py-2 text-sm transition ${
                        checked
                          ? "border-blue-200 bg-blue-50 text-blue-700"
                          : "border-slate-200 bg-white text-slate-600 hover:bg-slate-50"
                      }`}
                    >
                      <input
                        type="checkbox"
                        checked={checked}
                        disabled={isBusy}
                        onChange={() => toggleTag(tag.id)}
                        className="accent-blue-600"
                      />

                      {tag.name}
                    </label>
                  );
                })}
              </div>
            )}
          </div>

          {/* Create Group and Tag */}
          <div className="grid gap-5 border-t border-slate-100 pt-5 lg:grid-cols-2">
            <form
              onSubmit={handleCreateGroup}
              className="space-y-3"
            >
              <h3 className="text-sm font-semibold text-slate-800">
                Create Group
              </h3>

              <input
                value={groupName}
                onChange={(event) =>
                  setGroupName(event.target.value)
                }
                placeholder="Group name"
                maxLength={100}
                required
                disabled={isBusy}
                className="h-10 w-full rounded-lg border border-slate-200 px-3 text-sm outline-none focus:border-blue-500 disabled:opacity-50"
              />

              <textarea
                value={groupDescription}
                onChange={(event) =>
                  setGroupDescription(event.target.value)
                }
                placeholder="Description (optional)"
                rows={2}
                disabled={isBusy}
                className="w-full rounded-lg border border-slate-200 px-3 py-2 text-sm outline-none focus:border-blue-500 disabled:opacity-50"
              />

              <button
                type="submit"
                disabled={isBusy || !groupName.trim()}
                className="inline-flex items-center gap-2 rounded-lg border border-slate-200 px-3 py-2 text-xs font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-50"
              >
                <Plus size={14} />
                {creatingGroup ? "Creating..." : "Create Group"}
              </button>
            </form>

            <form
              onSubmit={handleCreateTag}
              className="space-y-3"
            >
              <h3 className="text-sm font-semibold text-slate-800">
                Create Tag
              </h3>

              <input
                value={tagName}
                onChange={(event) =>
                  setTagName(event.target.value)
                }
                placeholder="Tag name"
                maxLength={50}
                required
                disabled={isBusy}
                className="h-10 w-full rounded-lg border border-slate-200 px-3 text-sm outline-none focus:border-blue-500 disabled:opacity-50"
              />

              <p className="text-xs text-slate-500">
                Examples: production, critical, database, internal.
              </p>

              <button
                type="submit"
                disabled={isBusy || !tagName.trim()}
                className="inline-flex items-center gap-2 rounded-lg border border-slate-200 px-3 py-2 text-xs font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-50"
              >
                <Plus size={14} />
                {creatingTag ? "Creating..." : "Create Tag"}
              </button>
            </form>
          </div>

          {/* Feedback */}
          {error && (
            <p
              role="alert"
              className="rounded-lg bg-red-50 p-3 text-sm text-red-600"
            >
              {error}
            </p>
          )}

          {success && (
            <p
              role="status"
              className="rounded-lg bg-emerald-50 p-3 text-sm text-emerald-700"
            >
              {success}
            </p>
          )}

          {/* Actions */}
          <div className="flex flex-wrap items-center justify-between gap-3 border-t border-slate-100 pt-5">
            <button
              type="button"
              onClick={() => void handleReload()}
              disabled={isBusy || loading}
              className="text-sm font-medium text-slate-500 hover:text-slate-800 disabled:opacity-50"
            >
              Reload metadata
            </button>

            <button
              type="button"
              onClick={() => void handleSave()}
              disabled={isBusy || loading}
              className="inline-flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-blue-700 disabled:opacity-50"
            >
              <Save size={15} />
              {saving ? "Saving..." : "Save Changes"}
            </button>
          </div>
        </div>
      )}
    </section>
  );
}

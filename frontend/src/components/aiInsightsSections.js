// Shared helpers for the AI Insights panel: section splitting and the
// section-level diff used by the compare view.

// Splits markdown output into its `## Heading` sections, preserving order.
// Content before the first `##` heading lands under the "" key.
export function splitSections(markdown) {
  const sections = new Map();
  let currentKey = "";
  let currentLines = [];
  for (const line of (markdown || "").split("\n")) {
    const heading = line.match(/^##\s+(.*)$/);
    if (heading) {
      sections.set(currentKey, currentLines.join("\n").trim());
      currentKey = heading[1].trim();
      currentLines = [];
    } else {
      currentLines.push(line);
      continue;
    }
  }
  sections.set(currentKey, currentLines.join("\n").trim());
  // Drop the pre-heading fragment if empty — it normally is, since the model
  // is asked to lead with a `## Summary` section.
  if (sections.get("") === "") {
    sections.delete("");
  }
  return sections;
}

// Classifies each shared section heading as added/removed/changed/same.
// `changed` means the trimmed bodies differ beyond whitespace-only noise.
function classifySections(fromSections, toSections) {
  const headings = [];
  const seen = new Set();
  for (const heading of fromSections.keys()) {
    headings.push({ heading, status: toSections.has(heading) ? "common" : "removed" });
    seen.add(heading);
  }
  for (const heading of toSections.keys()) {
    if (!seen.has(heading)) {
      headings.push({ heading, status: "added" });
    }
  }
  return headings.map(({ heading, status }) => {
    if (status === "common") {
      const from = fromSections.get(heading);
      const to = toSections.get(heading);
      const same = normalizeBody(from) === normalizeBody(to);
      return { heading, from, to, status: same ? "same" : "changed" };
    }
    return { heading, from: fromSections.get(heading) ?? null, to: toSections.get(heading) ?? null, status };
  });
}

// Whitespace-normalized comparison: collapses runs of whitespace and trims,
// so reflowed-but-identical text is not flagged as changed.
function normalizeBody(body) {
  return (body || "").replace(/\s+/g, " ").trim();
}

export function diffSections(fromMarkdown, toMarkdown) {
  const differences = classifySections(
    splitSections(fromMarkdown || ""),
    splitSections(toMarkdown || ""),
  );
  return {
    sections: differences,
    // A change worth surfacing: any added/removed/changed section. Pure
    // whitespace differences count as same.
    hasChanges: differences.some((item) => item.status !== "same"),
  };
}

// Comparison status → label/color class mapping for the UI.
export const SECTION_STATUS_META = {
  changed: { label: "Changed", className: "diff-changed" },
  added: { label: "Added", className: "diff-added" },
  removed: { label: "Removed", className: "diff-removed" },
  same: { label: "Unchanged", className: "diff-same" },
};

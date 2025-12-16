# Plugin Trust Model (Design Specification)

**Status**: Design phase (not implemented)
**Target Version**: v0.3.0 or later
**Last Updated**: 2025-12-16

---

## Overview

This document defines a future trust model for marketplace plugins in Plum. The goal is to provide users with confidence indicators while maintaining the current "data-only, never execute" security posture.

**Core Principle**: Plugin metadata is display data, not executable instructions. This model classifies and validates that data, but never grants plugins execution privileges.

---

## Trust Levels

### 1. Unverified (Default)

**Status**: No validation beyond basic schema checks
**Visual Indicator**: None (no badge)
**Criteria**: Any plugin from any marketplace

**Behavior**:
- Displayed in search results without special treatment
- No warnings or alerts
- User decides whether to trust based on source/marketplace

**Risk**: Highest potential for misleading or malicious metadata

### 2. Verified

**Status**: Passed automated validation pipeline
**Visual Indicator**: `[Verified]` badge in TUI
**Criteria**:
- Schema validation (all required fields present)
- Content classification (metadata vs. instructions)
- Size limits (description < 500 chars, name < 100 chars)
- No known phishing patterns in URLs
- Marketplace repo has activity within last 6 months

**Behavior**:
- Displayed with visual trust indicator
- Cached verification status (expires in 7 days)
- Re-validated on cache expiry

**Risk**: Medium - automated checks reduce obvious threats

### 3. Trusted (Future)

**Status**: Manual review + community vetting
**Visual Indicator**: `[Trusted]` badge with checkmark
**Criteria**:
- All "Verified" criteria met
- Manual review by Plum maintainers or community moderators
- Publisher identity verified (GitHub org, domain ownership)
- Marketplace has established reputation (6+ months, 50+ stars)
- Active maintenance (commits within last 3 months)

**Behavior**:
- Displayed prominently in search results
- Optional filter: "Show only trusted plugins"
- Trust status cached indefinitely (until revocation)

**Risk**: Lowest - multiple layers of validation

---

## Validation Pipeline (Conceptual)

### Phase 1: Schema Validation

**Timing**: On manifest fetch (before caching)
**Enforcement**: Strict (invalid manifests rejected)

**Checks**:
1. JSON structure matches `MarketplaceManifest` schema
2. Required fields present: `name`, `plugins`, `metadata`
3. Plugin entries have `name`, `source`, `description`
4. Fields conform to type constraints (strings, arrays, booleans)

**Failure Mode**: Skip marketplace, log warning, do not cache

### Phase 2: Content Classification

**Timing**: On manifest fetch (after schema validation)
**Enforcement**: Advisory (warnings only, does not block)

**Checks**:
1. **Detect instruction-like patterns** in descriptions:
   - Command syntax (e.g., "run this script", "execute the following")
   - Markdown code blocks with shell commands
   - Keywords: "sudo", "rm -rf", "curl | bash", etc.

2. **Detect phishing indicators** in URLs:
   - Typosquatting (e.g., "g1thub.com" vs. "github.com")
   - Suspicious TLDs (.tk, .ml, .ga, etc.)
   - IP addresses instead of domains

3. **Detect anomalous metadata**:
   - Description length > 500 characters
   - Name contains special characters beyond allowlist
   - Keywords list > 20 entries

**Failure Mode**: Flag as `[Warning]` in TUI, user can still view

### Phase 3: Validation Pipeline Integration

```
Fetch GitHub Manifest
        |
        v
   Schema Valid? --------NO-------> Skip + Log
        |
       YES
        |
        v
  Content Classification
        |
        +---> Suspicious Patterns? --YES--> Mark "Warning"
        |
        +---> All Checks Pass? -----YES--> Mark "Verified"
        |
        v
   Save to Cache (with trust metadata)
        |
        v
   Display in TUI (with badge/warning)
```

---

## Trust Metadata (Cache Extension)

Extend `CacheEntry` to include trust information:

```go
type CacheEntry struct {
    Manifest     *MarketplaceManifest `json:"manifest"`
    FetchedAt    time.Time            `json:"fetchedAt"`
    Source       string               `json:"source"`
    TrustStatus  string               `json:"trustStatus"`  // "unverified", "verified", "trusted", "warning"
    ValidationAt time.Time            `json:"validationAt"` // Last validation timestamp
    Warnings     []string             `json:"warnings"`     // List of issues found
}
```

**Trust Status TTL**:
- `unverified`: Re-check on cache expiry (24 hours)
- `verified`: Re-check every 7 days
- `trusted`: Persist until explicit revocation
- `warning`: Re-check every 3 days (in case false positive)

---

## Future Agent Validator (Conceptual)

### Purpose
An optional automated agent that reviews plugin metadata for suspicious patterns. This agent is **advisory only** and has no write access.

### Capabilities
- Analyze plugin descriptions for instruction-like language
- Flag potential prompt injection patterns
- Detect social engineering indicators
- Suggest trust level based on heuristics

### Constraints
- **Read-only access**: Cannot modify metadata or cache
- **No execution**: Never interprets or evaluates plugin content
- **Advisory output**: Results displayed to user, never auto-enforced
- **Isolated**: Runs in separate process, cannot affect Plum runtime

### Example Workflow
1. User triggers manual review: `plum review-plugin <name>`
2. Agent fetches cached manifest
3. Agent analyzes description, keywords, URLs
4. Agent outputs report:
   ```
   Plugin: claude-code-plugin-xyz
   Trust Status: Unverified
   Warnings:
   - Description contains command syntax: "run npm install"
   - Repository URL uses suspicious TLD: .tk
   Recommendation: Exercise caution before installing
   ```
4. User decides whether to proceed

### Non-Goals
- **No auto-blocking**: Agent cannot prevent plugin display
- **No auto-promotion**: Agent cannot upgrade trust level to "Trusted"
- **No network access**: Agent only analyzes cached data
- **No LLM forwarding**: Plugin text is NOT sent to external AI APIs

---

## Verification Workflow (Manual "Trusted" Promotion)

### Process
1. Plugin maintainer submits verification request (GitHub issue template)
2. Plum maintainer reviews:
   - Marketplace repository health
   - Plugin metadata quality
   - Publisher identity (GitHub org, domain ownership)
   - Community reputation (stars, forks, issues)
3. Maintainer updates "trusted registry" file (e.g., `trusted-marketplaces.json`)
4. Plum loads trusted registry on startup
5. Plugins from trusted marketplaces display `[Trusted]` badge

### Revocation
1. Community reports issue (security, spam, abandoned)
2. Maintainer investigates
3. If confirmed, remove from trusted registry
4. Next Plum update removes `[Trusted]` badge

**Transparency**: All trust decisions logged in public GitHub issues

---

## Implementation Roadmap

**Note**: Version numbers are aspirational and may change based on community feedback and maintainer capacity.

### Phase 1: Schema Validation (v0.3.0)
- [ ] Define strict JSON schema for `MarketplaceManifest`
- [ ] Implement validation on manifest fetch
- [ ] Add unit tests for schema edge cases
- [ ] Reject invalid manifests with clear error messages

### Phase 2: Content Classification (v0.4.0)
- [ ] Implement pattern detection for instruction-like text
- [ ] Add phishing URL checks (typosquatting, suspicious TLDs)
- [ ] Display `[Warning]` badge for flagged plugins
- [ ] Add `--show-warnings` flag to list all warnings

### Phase 3: Verified Badge (v0.5.0)
- [ ] Extend `CacheEntry` with trust metadata
- [ ] Implement "Verified" criteria checks
- [ ] Display `[Verified]` badge in TUI
- [ ] Add trust status to detail view

### Phase 4: Trusted Registry (v0.6.0)
- [ ] Create `trusted-marketplaces.json` in repo
- [ ] Implement trust registry loader
- [ ] Add verification request process (GitHub issue template)
- [ ] Display `[Trusted]` badge for promoted plugins

### Phase 5: Agent Validator (v1.0.0+)
- [ ] Design agent review API (read-only interface)
- [ ] Implement heuristic pattern detection
- [ ] Add `plum review-plugin` command
- [ ] Document agent limitations and non-goals

---

## Security Considerations

### What This Model Does NOT Do

1. **No plugin execution**: Trust levels do not grant execution privileges
2. **No auto-install**: User must manually run install commands
3. **No agent instruction injection**: Plugin text is never forwarded to LLMs
4. **No false sense of security**: "Verified" means "passed automated checks", not "guaranteed safe"

### Defense in Depth

This model complements existing mitigations:
- Filename sanitization (prevents path traversal)
- Atomic cache writes (prevents race conditions)
- HTTP size limits (prevents DoS)
- User-only permissions (limits filesystem access)

Trust levels add **transparency**, not **enforcement**.

---

## Open Questions

1. **Who maintains the trusted registry?**
   - Option A: Plum maintainers only (centralized)
   - Option B: Community moderators (distributed)
   - Option C: Hybrid (maintainers + elected moderators)

2. **Should "Warning" plugins be hidden by default?**
   - Pro: Protects users from obvious threats
   - Con: May cause false positives, limits discovery
   - Proposed: Display with warning, add `--hide-warnings` flag

3. **How to handle marketplace disputes?**
   - Example: Marketplace claims "Trusted" status is unfair
   - Process: Public GitHub issue, transparent review, appeal mechanism

4. **Should trust status be exportable?**
   - Use case: Share trusted plugin lists between teams
   - Format: JSON export/import of trust metadata

---

## References

- [OWASP Top 10 2021](https://owasp.org/Top10/)
- [Prompt Injection Research (2023)](https://arxiv.org/abs/2302.12173)
- [Supply Chain Attacks (NIST)](https://csrc.nist.gov/publications/detail/sp/800-161/rev-1/final)
- [Metadata as Attack Surface (Simon Willison)](https://simonwillison.net/2023/Apr/14/worst-that-can-happen/)

---

**Next Steps**:
1. Gather community feedback on this design (GitHub discussion)
2. Prioritize phases based on user needs
3. Implement Phase 1 (Schema Validation) in v0.3.0

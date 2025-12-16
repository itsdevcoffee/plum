# Security Policy

## Project Overview

Plum is a terminal user interface (TUI) for discovering and browsing Claude Code marketplace plugins. It fetches plugin metadata from GitHub repositories and displays it in a searchable interface.

**What Plum does:**
- Fetches marketplace manifests (JSON) from GitHub
- Caches plugin metadata locally
- Displays plugin information in a TUI
- Copies install commands to clipboard

**What Plum does NOT do:**
- Execute plugin code
- Evaluate or interpret plugin content
- Forward plugin text as instructions to any agent or LLM
- Auto-install plugins or modify Claude Code configuration

## Threat Model

### Untrusted Inputs

Plum treats the following as untrusted:

1. **Marketplace registry data** (GitHub raw content)
   - Marketplace manifests (`marketplace.json`)
   - Plugin metadata (names, descriptions, authors, categories, keywords, tags)
   - HTTP responses from GitHub

2. **User-controlled inputs**
   - Search queries
   - Marketplace names (indirectly via registry)

### Trusted Components

1. **Local filesystem** within user home directory (`~/.plum/`)
2. **Compiled Plum binary** (built from source or official releases)
3. **Hardcoded GitHub raw content base URL** (`https://raw.githubusercontent.com`)

## Trust Boundaries

### Critical Boundaries

1. **Plugin metadata is data only, never instructions**
   - All plugin fields (name, description, keywords) are treated as display text
   - No plugin content is executed, evaluated, or interpreted as code
   - No plugin text is forwarded to AI agents or LLMs as system instructions

2. **Filesystem operations are strictly controlled**
   - Marketplace names are validated before filesystem use
   - Only alphanumeric, dash, underscore, and dot characters allowed
   - Path traversal sequences (`..`, `/`, `\`) are rejected
   - All cache writes are atomic (temp file + rename)

3. **Network operations are limited**
   - HTTP response bodies capped at 10 MB
   - 30-second timeouts on all requests
   - Retry logic only for transient failures (5xx, 429, network errors)
   - Client errors (4xx) are not retried

## Known Risks

### Supply Chain Attacks

**Risk**: Malicious marketplace entries could contain:
- Misleading plugin descriptions
- Phishing URLs in homepage/repository fields
- Malicious install commands (displayed to user for manual execution)

**Impact**: User trust and potential social engineering vectors

### Prompt Injection via Plugin Metadata

**Risk**: Plugin descriptions or names crafted to manipulate future AI agent processing if metadata is ever forwarded as context.

**Context**: Research on prompt injection (e.g., [Adversarial Machine Learning](https://arxiv.org/abs/2302.12173)) shows that even display-only text can pose risks if later processed by LLMs.

**Current Status**: Plum does not forward plugin text to any AI system. This risk is acknowledged for future-proofing.

### Registry Compromise

**Risk**: If a marketplace's GitHub repository is compromised, attackers could inject malicious metadata.

**Impact**: Integrity of displayed information, potential social engineering

### Denial of Service

**Risk**: Malformed or oversized responses could cause:
- Memory exhaustion (mitigated by 10 MB limit)
- Cache pollution (mitigated by 24-hour TTL and atomic writes)
- UI hangs (mitigated by 30-second HTTP timeout)

## Mitigations Implemented

### Filename Sanitization
- `validateMarketplaceName()` enforces strict character allowlist
- Blocks path traversal (`..`), separators (`/`, `\`), and special characters
- Maximum name length: 100 characters

### Cache Safety
- Atomic writes using temp file + rename pattern
- User-only permissions (0700 for directories, 0600 for files)
- 24-hour cache TTL to limit stale data exposure

### Network Limits
- 10 MB HTTP response body limit (enforced via `io.LimitReader`)
- 30-second HTTP timeout per request
- 3 retry attempts with exponential backoff (1s, 2s, 4s)
- Transient-only retry logic (5xx, 429, network errors)

### HTTP Best Practices
- Singleton HTTP client for connection reuse
- Context-based request cancellation
- User-Agent header ("plum-marketplace-browser/0.2.0")
- Connection pooling (10 idle connections, 5 per host)

### CI/CD Hardening
- golangci-lint enforces security best practices
- gosec static analysis for vulnerability detection
- All PRs require passing CI checks
- Automated testing for cache and network operations

## Future Mitigations (Planned)

These features are NOT yet implemented but are under consideration:

### Registry Signing/Verification
- Cryptographic signatures for marketplace manifests
- Publisher identity verification via GPG or similar
- Trust chain from registry to individual plugins

### Plugin Validation Pipeline
- Schema validation for marketplace manifests
- Content classification (metadata vs. instructions)
- Automated checks for phishing indicators

### Verified Marketplace Concept
- Tiered trust levels (Unverified, Verified, Trusted)
- Manual review process for "Verified" status
- Visual indicators in TUI for trust level

### Agent-Based Review (Advisory Only)
- Automated analysis of plugin metadata
- Pattern detection for suspicious content
- Read-only access, never modifies data
- Results displayed as warnings, not enforcement

See `docs/plugin-trust-model.md` for detailed design proposals.

## Reporting Vulnerabilities

We take security seriously. If you discover a security issue:

1. **GitHub Issues**: For non-critical issues, open a public issue at https://github.com/itsdevcoffee/plum/issues
2. **Private Report**: For critical vulnerabilities, email the maintainers (see GitHub profile for contact)
3. **Provide details**: Steps to reproduce, impact assessment, suggested fix if available

**Response Timeline**:
- Acknowledgment: Within 3 business days
- Initial assessment: Within 7 business days
- Fix timeline: Depends on severity (critical issues prioritized)

## Security Best Practices for Users

1. **Review install commands** before executing them
   - Plum displays commands but does not execute them
   - Verify repository URLs and marketplace sources

2. **Use official marketplaces** when possible
   - Prefer well-known, community-vetted marketplaces
   - Check repository activity and community engagement

3. **Keep Plum updated**
   - Security patches are released in new versions
   - Check for updates regularly via GitHub releases

4. **Report suspicious plugins**
   - If you encounter malicious or misleading metadata, report to both:
     - The marketplace maintainer (via GitHub issues on their repo)
     - Plum maintainers (if systemic issue)

## License

This security policy is part of the Plum project and is licensed under the MIT License.

---

**Last Updated**: 2025-12-16
**Version**: 1.0
**Contact**: See [README.md](README.md) for project maintainers

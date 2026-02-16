#!/usr/bin/env python3
"""Tests for the ecosystem scorer module."""

import unittest

from scorer import (
    infer_tags,
    compute_tag_score,
    normalize_popularity,
    deduplicate,
    score_item,
)


class TestInferTags(unittest.TestCase):
    """Test keyword inference with word boundary matching."""

    def test_go_not_inferred_from_django(self):
        """'go' should NOT match inside 'Django'."""
        item = {"description": "Django rest framework", "tags": []}
        tags = infer_tags(item)
        self.assertNotIn("go", tags)

    def test_go_inferred_from_golang(self):
        item = {"description": "Golang web framework", "tags": []}
        tags = infer_tags(item)
        self.assertIn("go", tags)

    def test_go_inferred_standalone(self):
        item = {"description": "A go library for HTTP", "tags": []}
        tags = infer_tags(item)
        self.assertIn("go", tags)

    def test_go_not_inferred_from_cargo(self):
        item = {"description": "Cargo build system", "tags": []}
        tags = infer_tags(item)
        self.assertNotIn("go", tags)

    def test_java_not_inferred_from_javascript(self):
        item = {"description": "JavaScript framework", "tags": []}
        tags = infer_tags(item)
        self.assertNotIn("java", tags)
        self.assertIn("javascript", tags)

    def test_git_not_inferred_from_digital(self):
        item = {"description": "Digital transformation tools", "tags": []}
        tags = infer_tags(item)
        self.assertNotIn("git", tags)

    def test_git_inferred_standalone(self):
        item = {"description": "Git tools for automation", "tags": []}
        tags = infer_tags(item)
        self.assertIn("git", tags)

    def test_auth_not_inferred_from_author(self):
        item = {"description": "Author biography tool", "tags": []}
        tags = infer_tags(item)
        self.assertNotIn("authentication", tags)

    def test_auth_inferred_standalone(self):
        item = {"description": "Auth middleware for Express", "tags": []}
        tags = infer_tags(item)
        self.assertIn("authentication", tags)

    def test_react_native_removes_react(self):
        """When both react and react-native match, react should be removed."""
        item = {"description": "React Native mobile development", "tags": []}
        tags = infer_tags(item)
        self.assertIn("react-native", tags)
        self.assertNotIn("react", tags)

    def test_react_without_native_stays(self):
        item = {"description": "React component library", "tags": []}
        tags = infer_tags(item)
        self.assertIn("react", tags)
        self.assertNotIn("react-native", tags)

    def test_skip_already_explicit(self):
        """Tags already in explicit tags should not be inferred."""
        item = {"description": "Python testing framework", "tags": ["python"]}
        tags = infer_tags(item)
        self.assertNotIn("python", tags)
        self.assertIn("testing", tags)

    def test_cli_inferred(self):
        item = {"description": "CLI tool for deployment", "tags": []}
        tags = infer_tags(item)
        self.assertIn("cli", tags)
        self.assertIn("deployment", tags)

    def test_rust_not_inferred_from_trust(self):
        item = {"description": "Trust verification system", "tags": []}
        tags = infer_tags(item)
        self.assertNotIn("rust", tags)

    def test_api_not_inferred_from_capital(self):
        """'api' should match case-insensitively."""
        item = {"description": "REST API endpoints", "tags": []}
        tags = infer_tags(item)
        self.assertIn("api", tags)

    def test_github_actions_variants(self):
        item = {"description": "GitHub Action for CI/CD", "tags": []}
        tags = infer_tags(item)
        self.assertIn("github-actions", tags)
        self.assertIn("ci-cd", tags)


class TestComputeTagScore(unittest.TestCase):
    """Test the geometric mean tag scoring."""

    def test_perfect_overlap(self):
        """All item tags match all stack tags → high score."""
        score, explicit, inferred = compute_tag_score(
            {"go", "cli"}, {"go", "cli"},
        )
        self.assertGreater(score, 90)
        self.assertEqual(sorted(explicit), ["cli", "go"])

    def test_no_overlap(self):
        score, explicit, inferred = compute_tag_score(
            {"python", "django"}, {"go", "cli"},
        )
        self.assertEqual(score, 0.0)
        self.assertEqual(explicit, [])

    def test_partial_overlap(self):
        score, explicit, inferred = compute_tag_score(
            {"go", "cli", "tui", "terminal"}, {"go", "cli"},
        )
        self.assertGreater(score, 0)
        self.assertLess(score, 100)

    def test_inferred_tags_discounted(self):
        """Inferred tags should contribute less than explicit."""
        score_explicit, _, _ = compute_tag_score(
            {"go", "cli"}, {"go", "cli"},
        )
        score_inferred, _, _ = compute_tag_score(
            set(), {"go", "cli"}, inferred_tags={"go", "cli"},
        )
        self.assertGreater(score_explicit, score_inferred)

    def test_empty_stack(self):
        score, _, _ = compute_tag_score({"go"}, set())
        self.assertEqual(score, 0.0)


class TestNormalizePopularity(unittest.TestCase):
    """Test log-scale popularity normalization."""

    def test_highest_gets_100(self):
        items = [{"stars": 1000}, {"stars": 100}, {"stars": 10}]
        normalize_popularity(items, "stars")
        self.assertAlmostEqual(items[0]["_pop_norm"], 100.0, places=1)
        self.assertGreater(items[1]["_pop_norm"], items[2]["_pop_norm"])

    def test_zero_stays_zero(self):
        items = [{"stars": 100}, {"stars": 0}]
        normalize_popularity(items, "stars")
        self.assertEqual(items[1]["_pop_norm"], 0.0)

    def test_recency_boost(self):
        # Use two items so the non-max item shows the boost (max caps at 100)
        items = [{"stars": 1000}, {"stars": 50}]
        normalize_popularity(items, "stars", "2026-02-15")
        boosted = items[1]["_pop_norm"]

        items2 = [{"stars": 1000}, {"stars": 50}]
        normalize_popularity(items2, "stars", "2024-01-01")
        unboosted = items2[1]["_pop_norm"]

        self.assertGreater(boosted, unboosted)


class TestDeduplicate(unittest.TestCase):
    """Test cross-source deduplication."""

    def test_keeps_highest_score(self):
        entries = [
            {"owner": "foo", "name": "bar", "score": 50, "source": "plugins"},
            {"owner": "foo", "name": "bar", "score": 80, "source": "agents"},
        ]
        result = deduplicate(entries)
        self.assertEqual(len(result), 1)
        self.assertEqual(result[0]["score"], 80)
        self.assertEqual(result[0]["source"], "agents")

    def test_also_found_in(self):
        entries = [
            {"owner": "foo", "name": "bar", "score": 80, "source": "agents"},
            {"owner": "foo", "name": "bar", "score": 50, "source": "plugins"},
        ]
        result = deduplicate(entries)
        self.assertEqual(result[0]["also_found_in"], ["plugins"])

    def test_no_duplicates(self):
        entries = [
            {"owner": "a", "name": "x", "score": 80, "source": "agents"},
            {"owner": "b", "name": "y", "score": 50, "source": "plugins"},
        ]
        result = deduplicate(entries)
        self.assertEqual(len(result), 2)

    def test_case_insensitive_keys(self):
        entries = [
            {"owner": "Foo", "name": "Bar", "score": 80, "source": "agents"},
            {"owner": "foo", "name": "bar", "score": 50, "source": "plugins"},
        ]
        result = deduplicate(entries)
        self.assertEqual(len(result), 1)


class TestScoreItem(unittest.TestCase):
    """Test the combined scoring function."""

    def test_hard_gate_zero_tags(self):
        """Items with zero matching tags should return None."""
        item = {"tags": ["python", "django"], "description": "Django web", "_pop_norm": 50}
        result = score_item(item, {"go", "cli"}, "stars")
        self.assertIsNone(result)

    def test_matching_item_scores(self):
        item = {"tags": ["go", "cli", "tui"], "description": "Terminal UI", "_pop_norm": 75}
        result = score_item(item, {"go", "cli", "tui"}, "stars")
        self.assertIsNotNone(result)
        score, explicit, inferred = result
        self.assertGreater(score, 0)
        self.assertIn("go", explicit)


if __name__ == "__main__":
    unittest.main()

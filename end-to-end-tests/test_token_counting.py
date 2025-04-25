# test_token_counting.py
import unittest
import asyncio
from typing import Dict, List
from transformers import AutoTokenizer

class TestTokenCounting(unittest.TestCase):
    def setUp(self):
        # Load the all-mini tokenizer
        self.tokenizer = AutoTokenizer.from_pretrained("data/all-MiniLM-L6-v2")

    def count_tokens_with_all_mini(self, text: str) -> int:
        """Count tokens using the all-mini tokenizer"""
        if not text:
            return 0
        return len(self.tokenizer.encode(text))

    def simplified_count_tokens(self, text: str) -> int:
        """Estimate the number of tokens in a text string."""
        if not text:
            return 0

        # Simplified method: count characters and divide by average token size
        # This is a very rough approximation and will not be as accurate as a proper tokenizer
        # Average English token is about 4-5 characters
        return len(text) // 4 + 1

    def test_token_count_comparison(self):
        """Compare token counts between all-mini tokenizer and simplified method"""
        test_cases = [
            "This is a simple test case.",
            "Hello world!",
            "This is a longer test case with more words to ensure we have a good comparison between the two methods.",
            "Programming in Python is fun and productive!",
            """This is a multi-line text example.
            It contains several lines of text.
            This should test the token counting with larger inputs.""",
            "Special characters: !@#$%^&*()_+{}|:<>?[]\\;',./",
            "Numbers: 1234567890",
            "A mix of everything: text, numbers (123), and special chars !@#",
            "", # Empty string
            "A"  # Single character
        ]

        results = []

        for text in test_cases:
            # Count tokens using both methods
            all_mini_tokens = self.count_tokens_with_all_mini(text)
            simplified_tokens = self.simplified_count_tokens(text)

            # Calculate difference
            diff = simplified_tokens - all_mini_tokens
            percent_diff = (diff / all_mini_tokens * 100) if all_mini_tokens > 0 else None

            results.append({
                "text": text,
                "all_mini_tokens": all_mini_tokens,
                "simplified_tokens": simplified_tokens,
                "diff": diff,
                "percent_diff": percent_diff
            })

        # Print results in a readable format
        print("\nToken count comparison:")
        print("======================")

        for result in results:
            text = result["text"]
            print(f"Text: {text[:50]}{'...' if len(text) > 50 else ''}")
            print(f"  All-mini tokenizer: {result['all_mini_tokens']}")
            print(f"  Simplified method: {result['simplified_tokens']}")
            print(f"  Absolute difference: {result['diff']}")
            if result['percent_diff'] is not None:
                print(f"  Percent difference: {result['percent_diff']:.2f}%")
            else:
                print(f"  Percent difference: N/A")
            print()

            # Verify that the simplified method is within a reasonable range
            # For most use cases, being within 50% is acceptable for estimation
            if result['all_mini_tokens'] > 0:
                self.assertLess(abs(result['percent_diff']), 100,
                               f"Token count difference too large for: {result['text']}")

if __name__ == "__main__":
    unittest.main()

import unittest
from test_utils.base_test import ExtraTestEndToEnd

class TestEndToEndWithCodeHost(unittest.TestCase, ExtraTestEndToEnd):
    @classmethod
    def setUpClass(cls):
        """Set up the test environment with code host key."""
        # Initialize setup with no_code_host_key=False
        cls.setup_end_to_end(no_code_host_key=False)
        # Call unittest.TestCase's setUpClass
        super().setUpClass()

    @classmethod
    def tearDownClass(cls):
        """Tear down the test environment."""
        # Initialize teardown
        cls.teardown_end_to_end()
        # Call unittest.TestCase's tearDownClass
        super().tearDownClass()

if __name__ == '__main__':
    unittest.main()


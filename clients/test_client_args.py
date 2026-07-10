import importlib.util
import runpy
import sys
import types
import unittest
from pathlib import Path


CLIENT_DIR = Path(__file__).resolve().parent


class ClientArgumentTests(unittest.TestCase):
    def test_password_with_user_text_does_not_replace_username(self):
        if importlib.util.find_spec("psutil") is None:
            sys.modules["psutil"] = types.ModuleType("psutil")

        arguments = [
            "SERVER=127.0.0.1",
            "PORT=35601",
            "USER=s01",
            "PASSWORD=USER_DEFAULT_PASSWORD",
            "INTERVAL=2",
            "NOTUSER=ignored",
        ]
        expected = {
            "SERVER": "127.0.0.1",
            "PORT": "35601",
            "USER": "s01",
            "PASSWORD": "USER_DEFAULT_PASSWORD",
            "INTERVAL": "2",
        }

        for filename in ("client-linux.py", "client-psutil.py"):
            with self.subTest(client=filename):
                namespace = runpy.run_path(str(CLIENT_DIR / filename))
                self.assertEqual(namespace["parse_cli_args"](arguments), expected)


if __name__ == "__main__":
    unittest.main()

import runpy
import sys
import threading
import time
import types
import unittest
from collections import namedtuple
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path
from unittest import mock


CLIENT_DIR = Path(__file__).resolve().parent


def load_client(filename):
    if filename == "client-psutil.py" and "psutil" not in sys.modules:
        try:
            __import__("psutil")
        except ImportError:
            sys.modules["psutil"] = types.ModuleType("psutil")
    return runpy.run_path(str(CLIENT_DIR / filename))


class ClientMetricTests(unittest.TestCase):
    def test_psutil_counter_reads_are_serialized_and_keep_nowrap_enabled(self):
        client = load_client("client-psutil.py")
        active = 0
        maximum_active = 0
        calls = []
        state_lock = threading.Lock()

        def fake_counters(*args, **kwargs):
            nonlocal active, maximum_active
            with state_lock:
                active += 1
                maximum_active = max(maximum_active, active)
                calls.append((args, kwargs))
            time.sleep(0.002)
            with state_lock:
                active -= 1
            return {}

        with mock.patch.object(client["psutil"], "net_io_counters", side_effect=fake_counters, create=True):
            with ThreadPoolExecutor(max_workers=12) as executor:
                list(executor.map(lambda _index: client["_get_net_io_counters"](), range(48)))

        self.assertEqual(maximum_active, 1)
        self.assertEqual(len(calls), 48)
        self.assertTrue(all(kwargs == {"pernic": True} for _args, kwargs in calls))

    def test_psutil_totals_exclude_virtual_interfaces(self):
        client = load_client("client-psutil.py")
        counters = namedtuple("Counters", "bytes_sent bytes_recv")
        values = {
            "eth0": counters(500, 1000),
            "ens5": counters(300, 700),
            "wlo1": counters(200, 400),
            "lo": counters(9000, 9000),
            "docker0": counters(8000, 8000),
            "veth123": counters(7000, 7000),
        }
        with mock.patch.object(client["psutil"], "net_io_counters", return_value=values, create=True):
            self.assertEqual(client["liuliang"](), (2100, 1000))

    def test_linux_totals_read_proc_and_exclude_virtual_interfaces(self):
        client = load_client("client-linux.py")
        proc_net_dev = """Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
  eth0: 1000 10 0 0 0 0 0 0 500 5 0 0 0 0 0 0
  ens5: 700 7 0 0 0 0 0 0 300 3 0 0 0 0 0 0
  wlo1: 400 4 0 0 0 0 0 0 200 2 0 0 0 0 0 0
    lo: 9000 9 0 0 0 0 0 0 9000 9 0 0 0 0 0 0
veth123: 8000 8 0 0 0 0 0 0 8000 8 0 0 0 0 0 0
"""
        with mock.patch("builtins.open", mock.mock_open(read_data=proc_net_dev)):
            self.assertEqual(client["liuliang"](), (2100, 1000))

    def test_interface_filter_keeps_wlo_and_excludes_virtual_interfaces(self):
        ignored = (
            "lo", "lo0", "Loopback Pseudo-Interface 1", "tun0", "docker0",
            "veth123", "br-test", "vmbr0", "vnet0", "kube-ipvs0",
        )
        included = ("wlo1", "eth0", "enp3s0", "bond0")

        for filename in ("client-linux.py", "client-psutil.py"):
            with self.subTest(client=filename):
                client = load_client(filename)
                predicate = client["is_ignored_network_interface"]
                self.assertTrue(all(predicate(name) for name in ignored))
                self.assertTrue(all(not predicate(name) for name in included))

    def test_linux_totals_keep_one_way_interfaces(self):
        client = load_client("client-linux.py")
        proc_net_dev = """Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
  eth0: 1000 10 0 0 0 0 0 0 0 0 0 0 0 0 0 0
  ens5: 0 0 0 0 0 0 0 0 300 3 0 0 0 0 0 0
"""
        with mock.patch("builtins.open", mock.mock_open(read_data=proc_net_dev)):
            self.assertEqual(client["liuliang"](), (1000, 300))

    def test_network_speed_starts_and_resets_at_zero(self):
        for filename in ("client-linux.py", "client-psutil.py"):
            with self.subTest(client=filename):
                client = load_client(filename)
                state = client["update_net_speed"].__globals__["netSpeed"]
                state.update({"clock": 0.0, "diff": 0.0, "avgrx": 0, "avgtx": 0, "netrx": 0, "nettx": 0})

                self.assertEqual(client["update_net_speed"](1000, 1000, 100.0), (0, 0))
                self.assertEqual(client["update_net_speed"](1400, 1300, 102.0), (200, 150))
                self.assertEqual(client["update_net_speed"](10, 1500, 103.0), (0, 200))
                self.assertEqual(client["update_net_speed"](5, 4, 104.0), (0, 0))

    def test_os_detection_uses_linux_distribution_id(self):
        os_release = 'NAME="Alpine Linux"\nID=alpine\nVERSION_ID=3.22\n'
        for filename in ("client-linux.py", "client-psutil.py"):
            with self.subTest(client=filename):
                client = load_client(filename)
                with mock.patch.object(client["platform"], "system", return_value="Linux"), \
                        mock.patch("builtins.open", mock.mock_open(read_data=os_release)):
                    self.assertEqual(client["get_os_name"](), "alpine")

    def test_os_detection_has_platform_fallbacks(self):
        psutil_client = load_client("client-psutil.py")
        with mock.patch.object(psutil_client["platform"], "system", return_value="Windows Server 2022"):
            self.assertEqual(psutil_client["get_os_name"](), "windows")

        linux_client = load_client("client-linux.py")
        with mock.patch.object(linux_client["platform"], "system", return_value="FreeBSD"):
            self.assertEqual(linux_client["get_os_name"](), "freebsd")

    def test_cpu_model_prefers_specific_model_and_has_vendor_fallback(self):
        linux_client = load_client("client-linux.py")
        linux_globals = linux_client["get_cpu_model"].__globals__
        with mock.patch.dict(linux_globals, {
            "get_cpuinfo_values": lambda: {"model name": "AMD EPYC 7B13"},
            "get_lscpu_info": lambda: {"vendor id": "AuthenticAMD", "architecture": "x86_64"},
        }):
            self.assertEqual(linux_client["get_cpu_model"](), "AMD EPYC 7B13")

        psutil_client = load_client("client-psutil.py")
        platform_module = psutil_client["platform"]
        uname = types.SimpleNamespace(processor="", machine="x86_64")
        with mock.patch.object(platform_module, "processor", return_value=""), \
                mock.patch.object(platform_module, "uname", return_value=uname), \
                mock.patch.object(platform_module, "machine", return_value="x86_64"), \
                mock.patch.object(platform_module, "platform", return_value="Linux GenuineIntel"):
            self.assertEqual(psutil_client["get_cpu_model"](), "GenuineIntel")


if __name__ == "__main__":
    unittest.main()

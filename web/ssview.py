#!/usr/bin/env python3
# coding: utf-8
# Update by : https://github.com/cppla/ServerStatus, Update date: 20211009
# 支持Python版本：2.7 to 3.9; requirements.txt: requests, PrettyTable
# 主要是为了受到CC attack时候方便查看机器状态

import os
import sys
import requests
import time
from prettytable import PrettyTable

scroll = True
clear = lambda: os.system('clear' if 'linux' in sys.platform or 'darwin' in sys.platform else 'cls')


def sscmd(address):
    while True:
        r = requests.get(
            url=address,
            headers={
                "User-Agent": "ServerStatus/20181203",
            }
        )
        jsonR = r.json()

        ss = PrettyTable(
            [
                "月流量 ↓|↑",
                "节点名",
                "位置",
                "在线时间",
                "负载",
                "网络 ↓|↑",
                "总流量 ↓|↑",
                "处理器",
                "内存",
                "硬盘"
            ],
        )
        for i in jsonR["servers"]:
            if i["online4"] is False and i["online6"] is False:
                ss.add_row(
                    [
                        '0.00G',
                        "%s" % i["name"],
                        "%s" % i["location"],
                        '-',
                        '-',
                        '-',
                        '-',
                        '-',
                        '-',
                        '-',
                    ]
                )
            else:
                ss.add_row(
                    [
                        "%.2fG|%.2fG" % (float(i["last_network_in"]) / 1024 / 1024 / 1024, float(i["last_network_out"]) / 1024 / 1024 / 1024),
                        "%s" % i["name"],
                        # "%s" % i["type"],
                        "%s" % i["location"],
                        "%s" % i["uptime"],
                        "%s" % (i["load_1"]),
                        "%.2fM|%.2fM" % (float(i["network_rx"]) / 1000 / 1000, float(i["network_tx"]) / 1000 / 1000),
                        "%.2fG|%.2fG" % (
                            float(i["network_in"]) / 1024 / 1024 / 1024, float(i["network_out"]) / 1024 / 1024 / 1024),
                        "%d%%" % (i["cpu"]),
                        "%d%%" % (float(i["memory_used"]) / i["memory_total"] * 100),
                        "%d%%" % (float(i["hdd_used"]) / i["hdd_total"] * 100),
                    ]
                )
        if scroll is True:
            clear()
        print(ss)
        time.sleep(1)


if __name__ == '__main__':
    default = 'https://tz.cloudcpp.com/json/stats.json'
    ads = sys.argv[1] if len(sys.argv) == 2 else default
    sscmd(ads)

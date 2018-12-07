#!/usr/bin/env python
# coding: utf-8
# Update by : https://github.com/cppla/ServerStatus
# 支持Python版本：2.7 to 3.5; requirements.txt: requests, PrettyTable
# 时间: 20180828
'''
maybe better by youself
'''

import os
import sys
import requests
import time
from prettytable import PrettyTable

# todo: 程序在非gui环境下目前有闪屏的bug
scroll = True
clear = lambda: os.system('clear' if 'linux' in sys.platform else 'cls')

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
                "Flight",
                "节点名",
                # "虚拟化",
                "位置",
                "在线时间",
                "负载",
                "网络",
                "流量",
                "处理器",
                "内存",
                "硬盘"
            ],
        )
        for i in jsonR["servers"]:
            ss.add_row(
                [
                    "%s" % 'MH361' if i["ip_status"] is True else 'MH370',
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
    ads = sys.argv[1] if len(sys.argv)==2 else default
    sscmd(ads)

#!/usr/bin/env python
# coding: utf-8
# Update by : https://github.com/cppla/ServerStatus
# 支持Python版本：2.7 to 3.8; requirements.txt: requests, PrettyTable
# 时间: 20210714， 主要是为了受到CC attack时候方便查看机器状态 && 程序在非gui环境下目前有闪屏的bug

"""
ServerStatus in console or shell, demo:
+--------+----------+--------+----------+------+-------------+-----------------+--------+------+------+
| Flight |  节点名  |  位置  | 在线时间 | 负载 |     网络    |       流量      | 处理器 | 内存 | 硬盘 |
+--------+----------+--------+----------+------+-------------+-----------------+--------+------+------+
| MH361  |  微软云  |  香港  |  50 天   | 0.11 | 0.00M|0.00M | 195.93G|173.64G |   0%   | 28%  | 29%  |
| MH361  |  腾讯云  |  香港  |  50 天   | 0.38 | 0.00M|0.00M |  51.89G|58.95G  |   2%   | 18%  | 34%  |
| MH361  |  微软云  | 新加坡 |  14 天   | 0.13 | 0.00M|0.00M |  17.28G|10.24G  |   1%   | 20%  |  5%  |
| MH361  | 甲骨文1  |  春川  |  61 天   | 0.0  | 0.00M|0.00M |   8.90G|10.20G  |   0%   | 25%  |  5%  |
| MH361  | 甲骨文2  |  春川  |  61 天   | 0.0  | 0.00M|0.00M |  11.82G|13.71G  |   0%   | 16%  |  5%  |
| MH370  | 甲骨文3  |  春川  |    -     |  -   |      -      |        -        |   -    |  -   |  -   |
| MH361  | cdn-aws1 |  edge  |   2 天   | 0.0  | 0.00M|0.00M |   5.33G|5.83G   |   0%   | 24%  | 10%  |
| MH361  | cdn-aws2 |  edge  |   1 天   | 0.03 | 0.01M|0.01M |   9.47G|4.97G   |   0%   | 17%  | 18%  |
| MH361  | cdn-aws3 |  edge  |   2 天   | 0.0  | 0.00M|0.00M |   3.73G|2.27G   |   0%   | 21%  |  8%  |
+--------+----------+--------+----------+------+-------------+-----------------+--------+------+------+
"""

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
            if i["online4"] is False and i["online6"] is False:
                ss.add_row(
                    [
                        'MH370',
                        "%s" % i["name"],
                        # "%s" % i["type"],
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
    ads = sys.argv[1] if len(sys.argv) == 2 else default
    sscmd(ads)
